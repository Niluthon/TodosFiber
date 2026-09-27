# Personal Task Tracker API

A small REST API for managing a personal list of tasks, written in Go with
[Fiber v3](https://gofiber.io/) and SQLite (via GORM).

## How to run

Requirements: Go 1.21+ (the module is pinned to the installed toolchain).

```bash
# install dependencies
go mod download

# run the API (listens on :8000 by default)
go run .

# run the tests
go test ./...
```

Configuration is done with environment variables:

| Variable | Default    | Description               |
|----------|------------|---------------------------|
| `PORT`   | `8000`     | HTTP port                 |
| `DB_DSN` | `todos.db` | SQLite database file path |

With Docker: `docker build -t todos-api . && docker run -p 8000:8000 todos-api`

### Endpoints

| Method | Path                     | Description                          |
|--------|--------------------------|--------------------------------------|
| POST   | `/tasks`                 | Create a task (returns 201)          |
| GET    | `/tasks`                 | List tasks, newest first             |
| GET    | `/tasks?status=done`     | Filter by status                     |
| GET    | `/tasks?page=1&limit=20` | Paginate the list                    |
| GET    | `/tasks/{id}`            | Get one task                         |
| PATCH  | `/tasks/{id}`            | Update title, status and/or due date |
| DELETE | `/tasks/{id}`            | Delete a task (returns 204)          |

Example:

```bash
curl -X POST localhost:8000/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Buy groceries","due_date":"2026-05-15"}'
```

## Architecture

- Each layer has its own responsibility and is independent from others. This makes the code more modular and easier to
  maintain.

## Technology choices

- **Fiber v3** was already the chosen framework: it is fast, has a good documentation, easy to use.
- **SQLite + GORM**: zero-setup persistence with easy local runs.
- Accelerates develompent, if perfomance needed in the future we can implement pure sql queries. Sqlite enough for now
  if its necessary we can swap it to
  Postgres/MySQL
  only means changing the driver in `database/database.go`.
- **go-playground/validator** Has similar structure with laravel validator.

## What I would improve with more time

- Return pagination metadata (total/pages) instead of a bare array.
- Add repository-level tests and database migrations (e.g. `golang-migrate`).
- DI container only if project starts growing. For now its overengineering. Init dependencies in main.go and pass it
  through constructor is enough for now.
- Cleaner folder structure for better readability for example (todo->data->DTO).
- Move routes to separate file (makes project cleaner) and group them by API versions.
- Refactor error handling to be more consistent and use custom error types with proper error messages.
- Add Fake data generator for testing purposes (PHP Faker alternative in Golang).
- Check naming conventions, fix them if needed to follow GoLang best practices.
- Add package [[github.com/joho/godotenv](https://github.com/joho/godotenv)] Move ENV variables to a `.env` file `.gitignore` it and add a `.env.example` file.

## Assumptions

- `due_date` is a date only (`YYYY-MM-DD`). An empty string in PATCH clears it.
- Titles are trimmed when checking the "done requires a title" rule, so a
  whitespace-only title is treated as missing.
- Status defaults to `pending` when omitted on create.
- There is a single user; no authentication is implemented.
