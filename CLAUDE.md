# CLAUDE.md — BodyMetrics Backend

> This file is auto-read by Claude Code at session start. It provides AI agents with everything needed to understand, navigate, and contribute to this repository.

---

## 1) Project Goal

REST API backend for the BodyMetrics Flutter app. Written in Go, uses MySQL for persistent storage, deployed via Docker. Provides:
- JWT-based authentication (register/login)
- API key validation for app-level security
- User profile CRUD
- Health metrics (weight, BMI) storage and retrieval

---

## 2) Architecture

**Stack:** Go 1.23 + gorilla/mux + MySQL 8.0 + JWT (HS256) + Docker

```
body-metrics-backend/
├── cmd/
│   └── server/
│       └── main.go                 # Entry point: config → DB → migrations → repos → handlers → router → serve
├── internal/
│   ├── config/
│   │   └── config.go              # Environment variable loading with fallback defaults
│   ├── db/
│   │   ├── mysql.go               # Connection pool (25 open, 5 idle, 5min lifetime)
│   │   └── migration.go           # Versioned, transactional migrations
│   ├── domain/
│   │   ├── user.go                # User struct (id, name, surname, gender, avatar, height, birth_of_date)
│   │   ├── metric.go              # UserMetric struct (id, user_id, date, weight, height, bmi, weight_diff, body_metric, created_at)
│   │   └── auth.go                # TokenRequest / TokenResponse DTOs
│   ├── handler/
│   │   ├── auth_handler.go        # POST /auth/register, POST /auth/login
│   │   ├── user_handler.go        # POST/GET/PATCH /users
│   │   ├── metric_handler.go      # POST/GET /users/{id}/metrics
│   │   └── response.go            # writeJSON, writeError helpers
│   ├── middleware/
│   │   ├── auth.go                # JWT validation middleware + token generation
│   │   └── apikey.go              # API key validation middleware (X-API-Key header)
│   └── repository/
│       ├── account_repo.go        # Account CRUD (email + password_hash)
│       ├── user_repo.go           # User CRUD (Create, GetByID, GetAll, Update)
│       └── metric_repo.go         # Metric CRUD (Create, GetByUserID)
├── Dockerfile                      # Multi-stage: golang:1.23-alpine → alpine:3.20
├── docker-compose.yml              # MySQL + API + PhpMyAdmin
├── .env / .env.example            # Environment configuration
├── go.mod / go.sum
└── README.md
```

**Layers:**
- `config/` → Environment variable loading
- `db/` → Connection pool + migrations
- `domain/` → Pure data structures (no logic)
- `repository/` → SQL queries (prepared statements only)
- `handler/` → HTTP request/response handling
- `middleware/` → Cross-cutting concerns (auth, API key)

---

## 3) API Endpoints

**Base Path:** `/api/v1`

### Public (No Auth Required, API Key Required)
| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | `/auth/register` | `AuthHandler.Register` | Register with email + password → returns JWT |
| POST | `/auth/login` | `AuthHandler.Login` | Login with email + password → returns JWT |

### Protected (JWT + API Key Required)
| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | `/users` | `UserHandler.Create` | Create user profile |
| GET | `/users` | `UserHandler.GetAll` | List all users |
| GET | `/users/{id}` | `UserHandler.GetByID` | Get user by ID |
| PATCH | `/users/{id}` | `UserHandler.Update` | Partial update user fields |
| POST | `/users/{id}/metrics` | `MetricHandler.Create` | Add health metric |
| GET | `/users/{id}/metrics` | `MetricHandler.GetByUserID` | Get all metrics for user |

### Response Format
```json
// Success
{"id": 1, "name": "John", ...}

// Error
{"error": "error message here"}
```

---

## 4) Authentication & Security

### Two-Layer Security

**Layer 1 — API Key (App-Level)**
- Header: `X-API-Key: <key>`
- Applied to ALL routes (public + protected)
- Prevents unauthorized clients from accessing the API
- Key stored in `API_KEY` environment variable
- If `API_KEY` env is empty, middleware is skipped (dev mode)

**Layer 2 — JWT Token (User-Level)**
- Header: `Authorization: Bearer <token>`
- Applied to protected routes only
- Algorithm: HS256
- Expiration: 30 days
- Claims: `account_id`, `email`, `exp`, `iat`
- Secret stored in `JWT_SECRET` environment variable

### Password Security
- Algorithm: bcrypt (default cost)
- Minimum length: 6 characters
- Stored as hash in `accounts.password_hash`

---

## 5) Database Schema

**Database:** MySQL 8.0 (`bodymetrics`)

### `schema_migrations` (Migration Tracking)
```sql
CREATE TABLE schema_migrations (
    version    VARCHAR(255) PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
)
```

### `accounts` (Authentication)
```sql
CREATE TABLE accounts (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
)
```

### `users` (Profiles)
```sql
CREATE TABLE users (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name          VARCHAR(100),
    surname       VARCHAR(100),
    gender        TINYINT,           -- 0=male, 1=female
    avatar        VARCHAR(50),       -- pr1, pr2, etc.
    height        INT,               -- cm
    birth_of_date VARCHAR(20),
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
)
```

### `user_metrics` (Health Metrics)
```sql
CREATE TABLE user_metrics (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id     BIGINT UNSIGNED NOT NULL,
    date        VARCHAR(20) NOT NULL,    -- dd-MM-yyyy (legacy display)
    weight      DOUBLE,                  -- kg
    height      INT NOT NULL,            -- cm
    bmi         DOUBLE NOT NULL,
    weight_diff DOUBLE,                  -- delta from previous
    body_metric VARCHAR(30),             -- BMI category enum name
    created_at  VARCHAR(30),             -- ISO8601 (canonical timestamp)
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
)
```

**Read order:** `ORDER BY created_at ASC, id ASC`

---

## 6) Migration System

File: `internal/db/migration.go`

- Versioned migrations in a Go slice
- Each migration runs in a transaction
- Automatic rollback on failure
- `schema_migrations` table tracks applied versions
- Migrations run on every server start (idempotent)

**Adding a new migration:**
1. Add a new struct to the `migrations` slice in `migration.go`
2. Use a sequential version prefix: `003_description`, `004_description`, etc.
3. Include both `up` SQL and a descriptive version name
4. Migrations must be idempotent where possible

---

## 7) Environment Configuration

**File:** `.env` (loaded by docker-compose and config.go)

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | MySQL host |
| `DB_PORT` | `3306` | MySQL port |
| `DB_USER` | `bodymetrics` | MySQL user |
| `DB_PASSWORD` | `bodymetrics_pass` | MySQL password |
| `DB_NAME` | `bodymetrics` | Database name |
| `MYSQL_ROOT_PASSWORD` | — | MySQL root password (docker) |
| `MYSQL_PORT` | `3306` | Exposed MySQL port (docker) |
| `PHPMYADMIN_PORT` | `8081` | PhpMyAdmin port (docker) |
| `JWT_SECRET` | `change-me-in-production` | JWT signing secret |
| `API_KEY` | — | API key for app-level auth (empty = disabled) |
| `PORT` | `8080` | API server port |

---

## 8) Docker Setup

**Services:**
1. **mysql** — MySQL 8.0 with health check, persistent volume
2. **api** — Go binary (multi-stage alpine build), depends on MySQL health
3. **phpmyadmin** — Database admin UI on port 8081

**Commands:**
```bash
# Start all services
docker compose up -d

# Rebuild after code changes
docker compose up -d --build api

# View logs
docker compose logs -f api

# Stop
docker compose down

# Reset database
docker compose down -v
```

---

## 9) Development Guidelines

### Code Conventions
- **Layered architecture:** domain → repository → handler → middleware
- **No business logic in handlers** — handlers only parse requests, call repos, write responses
- **Prepared statements only** — never string-concatenate SQL
- **All fields nullable** (pointer types in Go) where column allows NULL
- **Repository methods** return `(result, error)` pairs
- **Handler errors** use `writeError(w, status, message)` helper
- **Domain structs** have JSON tags matching Flutter model field names

### Adding a New Endpoint
1. Add domain struct to `internal/domain/` if needed
2. Add repository method to `internal/repository/`
3. Add handler method to `internal/handler/`
4. Register route in `cmd/server/main.go`
5. Add migration if new table/column needed

### Adding a New Migration
1. Open `internal/db/migration.go`
2. Add new entry to `migrations` slice with next version number
3. Write SQL in the `up` field
4. Restart server — migration runs automatically

### Security Checklist
- Never commit real secrets to `.env` (use `.env.example` for templates)
- Always use parameterized queries
- Validate input lengths and types in handlers
- Use bcrypt for password hashing (never plain text)
- Keep `JWT_SECRET` and `API_KEY` strong in production

---

## 10) Quick Debug Guide

### "API returns 401 Unauthorized"
1. Check `Authorization: Bearer <token>` header is present
2. Verify token hasn't expired (30-day TTL)
3. Check `JWT_SECRET` matches between token generation and validation

### "API returns 403 Forbidden"
1. Check `X-API-Key` header is present and matches `API_KEY` env variable
2. If `API_KEY` env is empty, middleware is disabled (dev mode)

### "Database connection failed"
1. Verify MySQL is running: `docker compose ps`
2. Check `.env` credentials match docker-compose environment
3. Ensure `DB_HOST=mysql` (docker) or `DB_HOST=localhost` (local)

### "Migration failed"
1. Check `schema_migrations` table for applied versions
2. Verify SQL syntax in the failing migration
3. Check if table/column already exists (idempotency)

### "Flutter app can't connect"
1. Android emulator: use `10.0.2.2` (not `localhost`)
2. iOS simulator: use `localhost` or `127.0.0.1`
3. Physical device: use machine's LAN IP
4. Check firewall allows port 8080

---

## 11) Dependency Reference

| Package | Version | Usage |
|---------|---------|-------|
| gorilla/mux | 1.8.1 | HTTP router |
| golang-jwt/jwt | 5.2.1 | JWT auth (HS256) |
| go-sql-driver/mysql | 1.8.1 | MySQL driver |
| golang.org/x/crypto | 0.41.0 | bcrypt password hashing |
