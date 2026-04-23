# Backend Master Class: Simple Bank

A simple banking backend service built with Go, PostgreSQL, Gin, and SQLC.

## Prerequisites

- Git
- Go 1.26 (project requires `go 1.26.1`)
- Docker
- Docker Compose
- `migrate` CLI if you want to run database migration targets from the Makefile

## Clone the repository

```bash
git clone https://github.com/Xavier-Tame/simple-bank.git
cd simple-bank
```

## Local development setup

The repository includes an `app.env` file with default local configuration:

- `DB_DRIVER=postgres`
- `DB_SOURCE=postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable`
- `SERVER_ADDRESS=0.0.0.0:8080`
- `TOKEN_SYMMETRIC_KEY=12345678901234567890123456789012`
- `ACCESS_TOKEN_DURATION=15m`

If you need to change the configuration, update `app.env` or provide matching environment variables.

### Option 1: Run with Docker Compose

```bash
docker compose up --build -d
```

This starts:

- `postgres` service with credentials `root` / `secret`
- `api` service built from the project Dockerfile

Then open `http://localhost:8080`.

### Option 2: Run locally using Makefile

Use the Makefile targets to start Postgres in Docker and run migrations manually:

```bash
make postgres
make createdb
make migrateup
make server
```

Notes:

- `make postgres` starts a PostgreSQL container on port `5432`
- `make createdb` creates the `simple_bank` database when using the standalone `postgres` target
- `make migrateup` runs migrations from `db/migration`
- `make server` starts the Go server with `main.go`

## Database migrations

The project uses SQL migration files in `db/migration`.

If `migrate` is not installed, use one of these:

```bash
go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Then run:

```bash
make migrateup
```

## Build and test

Run all tests:

```bash
make test
```

Generate SQL code with SQLC:

```bash
make sqlc
```

Generate mock DB code:

```bash
make mock
```

## Notes from the project

- The app loads configuration from `app.env` with Viper.
- `main.go` connects to the database and starts the API server.
- `Dockerfile` builds the app and includes the `migrate` binary for containerized startup.
- `docker-compose.yml` configures the API and Postgres services on a shared `bank-network`.
