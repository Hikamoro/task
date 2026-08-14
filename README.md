# Task Management API

Веб-API для управления задачами внутри команд: пользователи, команды с ролями,
задачи с версионированием (оптимистичная блокировка), история изменений,
комментарии и статистика. Реализовано на Go.

## Стек

- **Язык / фреймворк:** Go 1.24, стандартный `net/http` (без сторонних роутеров), `database/sql`
- **БД:** MySQL 8 (миграции — обычные `.sql`-файлы, применяются при старте)
- **SQL-запросы:** [sqlc](https://sqlc.dev) — типизированный код по SQL
- **Кеш:** Redis (списки задач команды с инвалидацией по версии)
- **Auth:** bcrypt + JWT (HS256)
- **Документация:** Swagger UI (генерируется [swag](https://github.com/swaggo/swag))
- **Тесты:** testcontainers-go (MySQL + Redis), интеграционные + сквозной HTTP-тест

## Структура

```
cmd/server/main.go       — точка входа, инициализация и graceful shutdown
internal/config/         — конфигурация из env
internal/db/             — подключение к MySQL + применение миграций
migrations/              — SQL-миграции
internal/sqlc/           — сгенерированный sqlc-код (queries.yaml)
internal/model/          — сущности, enum-ы, ошибки домена
internal/repository/     — репозиторий поверх sqlc + транзакции
internal/service/        — бизнес-логика, права доступа, отчёты
internal/cache/          — кеш Redis с инвалидацией
internal/auth/           — bcrypt + JWT
internal/httpapi/        — роутер, middleware, handlers
docs/                    — сгенерированный Swagger
```

## Запуск

### Docker Compose (всё сразу)

```bash
docker compose up --build
# API:      http://localhost:8080
# Swagger:  http://localhost:8080/swagger/index.html
```

### Локально (без Docker для приложения)

Нужны запущенные MySQL и Redis:

```bash
docker run -d --name task-mysql -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=task \
  -p 3306:3306 mysql:8.0
docker run -d --name task-redis -p 6379:6379 redis:7-alpine

go run ./cmd/api
```

## Конфигурация (env)

| Переменная | По умолчанию | Описание |
|---|---|---|
| `HTTP_ADDR` | `:8080` | адрес HTTP-сервера |
| `DB_DSN` | `root:root@tcp(localhost:3306)/task?...` | DSN MySQL |
| `REDIS_ADDR` | `localhost:6379` | адрес Redis |
| `REDIS_PASSWORD` | — | пароль Redis |
| `REDIS_DB` | `0` | номер БД Redis |
| `JWT_SECRET` | `dev-secret-change-me` | секрет JWT |
| `JWT_TTL` | `24h` | срок жизни токена |
| `CACHE_TTL` | `5m` | TTL кеша списков задач |
| `MIGRATE_ON_START` | `true` | применять миграции при старте |
| `RATE_LIMIT_RPS` | `10` | лимит запросов в секунду на IP |
| `RATE_LIMIT_BURST` | `30` | burst лимита |
| `MAX_BODY_BYTES` | `1MiB` | макс. размер тела запроса |

## Модель прав доступа

- **owner** команды — полный доступ: управление участниками, изменение ролей,
  удаление задач, статистика.
- **admin** — как owner, но не может менять роли других участников и удалять команду.
- **member** — создаёт/редактирует задачи и комментарии, не видит статистику.
- Пользователь вне команды не видит её задачи вообще.
- JWT содержит `user_id`; `team_id` в путях/запросах проверяется на членство.

## API (кратко)

| Метод | Путь | Доступ |
|---|---|---|
| POST | `/api/v1/register` | открытый |
| POST | `/api/v1/login` | открытый |
| POST | `/api/v1/teams` | авторизованный |
| GET | `/api/v1/teams` | авторизованный (мои команды) |
| GET | `/api/v1/teams/{id}/members` | участник команды |
| POST | `/api/v1/teams/{id}/invite` | owner/admin |
| PATCH | `/api/v1/teams/{id}/members/{uid}/role` | owner |
| DELETE | `/api/v1/teams/{id}/members/{uid}` | owner |
| GET | `/api/v1/teams/{id}/stats` | owner/admin |
| POST | `/api/v1/tasks?team_id=` | участник команды |
| GET | `/api/v1/tasks?team_id=` | участник команды (кеш Redis) |
| PUT | `/api/v1/tasks/{id}` | участник команды (version → 409 при конфликте) |
| GET | `/api/v1/tasks/{id}/history` | участник команды |
| POST | `/api/v1/tasks/{id}/comments` | участник команды |
| GET | `/api/v1/tasks/{id}/comments` | участник команды |
| GET | `/swagger/index.html` | открытый |

Полное описание схемы — в Swagger UI.

### Примеры (curl)

```bash
# Регистрация и вход
curl -s -X POST localhost:8080/api/v1/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"a@test.com","password":"secret123","name":"Alice"}'
TOKEN=$(curl -s -X POST localhost:8080/api/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"a@test.com","password":"secret123"}' | jq -r .token)

# Команда
curl -s -X POST localhost:8080/api/v1/teams -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"Dev"}'

# Задача и список
curl -s -X POST 'localhost:8080/api/v1/tasks?team_id=1' -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"title":"Write README"}'
curl -s 'localhost:8080/api/v1/tasks?team_id=1' -H "Authorization: Bearer $TOKEN"

# Обновление с версией (409 при устаревшей версии)
curl -s -X PUT localhost:8080/api/v1/tasks/1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"title":"x","status":"done","version":1}'

# Статистика
curl -s localhost:8080/api/v1/teams/1/stats -H "Authorization: Bearer $TOKEN"
```

## Оптимистичная блокировка

Каждая задача имеет монотонно растущий `version`. Запись через `PUT` обязана
передавать текущую версию; если в БД версия выше — возвращается `409 Conflict`.
История изменений фиксируется в `task_history` при каждом изменении полей.

## Тесты

Требуется запущенный Docker. Тесты поднимают MySQL и Redis через testcontainers.

```bash
go test ./... -count=1
```

Покрытие:
- `TestTeamStatsReport` — интеграционный тест SQL-отчёта (метрики за 30 дней, топ участников).
- `TestTaskAccessControl` — права доступа по ролям (owner/admin/member/вне команды).
- `TestOptimisticLockConflict` — версионирование: устаревшая версия → конфликт.
- `TestTaskHistoryRecording` — история изменений задач.
- `TestRoleManagement` — приглашение, смена роли, исключение.
- `TestAPIEndToEnd` — сквозной HTTP-тест: auth → команда → задачи → кеш → статистика → Swagger.

## Прочее

- Регенерация sqlc: `sqlc generate` (после правок `internal/sqlc/queries/`).
- Регенерация Swagger: `swag init -g cmd/server/main.go -o docs`.
