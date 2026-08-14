package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"task/internal/model"
)

type TaskCache struct {
	rdb *redis.Client
	ttl time.Duration
}

func New(rdb *redis.Client, ttl time.Duration) *TaskCache {
	return &TaskCache{rdb: rdb, ttl: ttl}
}

func verKey(teamID int64) string {
	return fmt.Sprintf("tasks_ver:%d", teamID)
}

func listKey(teamID, ver int64, fingerprint string) string {
	return fmt.Sprintf("tasks:%d:%d:%s", teamID, ver, fingerprint)
}

// Fingerprint builds a stable cache key fragment from the filter set.
func Fingerprint(status *model.TaskStatus, assigneeID *int64, limit, offset int32) string {
	h := sha256.New()
	fmt.Fprintf(h, "s=%v;", status)
	if assigneeID != nil {
		fmt.Fprintf(h, "a=%d;", *assigneeID)
	} else {
		fmt.Fprintf(h, "a=;")
	}
	fmt.Fprintf(h, "l=%d;o=%d", limit, offset)
	return hex.EncodeToString(h.Sum(nil))
}

// Bump invalidates all cached task lists for a team.
func (c *TaskCache) Bump(ctx context.Context, teamID int64) error {
	return c.rdb.Incr(ctx, verKey(teamID)).Err()
}

// Version returns the current cache version for a team.
func (c *TaskCache) Version(ctx context.Context, teamID int64) (int64, error) {
	v, err := c.rdb.Get(ctx, verKey(teamID)).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return v, nil
}

func (c *TaskCache) Get(ctx context.Context, teamID, ver int64, fingerprint string) (model.TaskList, bool, error) {
	raw, err := c.rdb.Get(ctx, listKey(teamID, ver, fingerprint)).Bytes()
	if err == redis.Nil {
		return model.TaskList{}, false, nil
	}
	if err != nil {
		return model.TaskList{}, false, err
	}
	var entry model.TaskList
	if err := json.Unmarshal(raw, &entry); err != nil {
		return model.TaskList{}, false, nil
	}
	return entry, true, nil
}

func (c *TaskCache) Set(ctx context.Context, teamID, ver int64, fingerprint string, entry model.TaskList) error {
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, listKey(teamID, ver, fingerprint), raw, c.ttl).Err()
}
