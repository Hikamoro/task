package service_test

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"task/internal/auth"
	"task/internal/cache"
	"task/internal/db"
	"task/internal/repository"
	"task/internal/service"
)

var (
	tsDB      *sql.DB
	tsRepo    *repository.Repository
	tsCache   *cache.TaskCache
	tsApp     *service.App
	tsDBName  = "task_test"
	testLog   = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	container testcontainers.Container
	redisC    testcontainers.Container
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	req := testcontainers.ContainerRequest{
		Image:        "mysql:8.0",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "root",
			"MYSQL_DATABASE":      tsDBName,
		},
		WaitingFor: wait.ForListeningPort("3306/tcp").WithStartupTimeout(2 * time.Minute),
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to start MySQL test container:", err)
		os.Exit(1)
	}
	container = c
	defer func() { _ = container.Terminate(context.Background()) }()

	host, err := c.Host(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "container host:", err)
		os.Exit(1)
	}
	port, err := c.MappedPort(ctx, "3306/tcp")
	if err != nil {
		fmt.Fprintln(os.Stderr, "container port:", err)
		os.Exit(1)
	}

	dsn := fmt.Sprintf("root:root@tcp(%s:%s)/%s?parseTime=true&multiStatements=true&charset=utf8mb4&loc=Local",
		host, port.Port(), tsDBName)

	database, err := repository.Open(dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect mysql:", err)
		os.Exit(1)
	}
	defer database.Close()

	// Wait until MySQL actually accepts connections (listening port can be up
	// before the server is ready).
	ready := false
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		if err := database.PingContext(ctx); err == nil {
			ready = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !ready {
		fmt.Fprintln(os.Stderr, "mysql did not become ready in time")
		os.Exit(1)
	}

	if err := db.MigrateUp(database); err != nil {
		fmt.Fprintln(os.Stderr, "run migrations:", err)
		os.Exit(1)
	}

	tsDB = database
	tsRepo = repository.New(database)
	tsApp = service.New(tsRepo, nil, auth.NewManager("test-secret-for-jwt", time.Hour), testLog)

	if err := startRedis(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "start redis:", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func startRedis(ctx context.Context) error {
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(2 * time.Minute),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return fmt.Errorf("start redis container: %w", err)
	}
	redisC = c

	host, err := c.Host(ctx)
	if err != nil {
		return err
	}
	port, err := c.MappedPort(ctx, "6379/tcp")
	if err != nil {
		return err
	}
	rdb := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%s", host, port.Port())})
	ready := false
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if err := rdb.Ping(ctx).Err(); err == nil {
			ready = true
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	if !ready {
		return fmt.Errorf("redis not ready")
	}
	tsCache = cache.New(rdb, 5*time.Minute)
	tsApp = service.New(tsRepo, tsCache, auth.NewManager("test-secret-for-jwt", time.Hour), testLog)
	return nil
}

// ---- helpers ----

func seedUser(t *testing.T, email, name string) int64 {
	t.Helper()
	u, err := tsApp.Register(context.Background(), email, "password123", name)
	if err != nil {
		t.Fatalf("seed user %s: %v", email, err)
	}
	return u.ID
}

func seedTeam(t *testing.T, ownerID int64, name string) int64 {
	t.Helper()
	team, err := tsApp.CreateTeam(context.Background(), ownerID, name)
	if err != nil {
		t.Fatalf("seed team %s: %v", name, err)
	}
	return team.ID
}

func createTask(ctx context.Context, t *testing.T, userID, teamID int64, title string, assignee *int64) int64 {
	t.Helper()
	task, err := tsApp.CreateTask(ctx, userID, teamID, title, "description", assignee)
	if err != nil {
		t.Fatalf("create task %s: %v", title, err)
	}
	return task.ID
}

// setTaskDone marks a task done with DB-side dates: the task is created
// createdDaysAgo days in the past and closed closeOffsetDays days after that.
// Doing the arithmetic in SQL keeps everything consistent with the session
// time zone.
func setTaskDone(t *testing.T, taskID int64, createdDaysAgo, closeOffsetDays int) {
	t.Helper()
	if _, err := tsDB.Exec(`
		UPDATE tasks
		SET created_at = DATE_SUB(CURRENT_TIMESTAMP, INTERVAL ? DAY),
		    status      = 'done',
		    closed_at   = DATE_SUB(CURRENT_TIMESTAMP, INTERVAL ? DAY) + INTERVAL ? DAY,
		    version     = version + 1
		WHERE id = ?`,
		createdDaysAgo, createdDaysAgo, closeOffsetDays, taskID,
	); err != nil {
		t.Fatalf("set task %d done: %v", taskID, err)
	}
}
