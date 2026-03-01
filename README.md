# BodyMetrics Backend API

Go + MySQL REST API for the BodyMetrics mobile app.  
BodyMetrics mobil uygulamasi icin Go + MySQL tabanli REST API servisi.

## 🚀 Project Overview / Proje Ozeti

**EN:** This service handles authentication, user profiles, and body metric history for the Flutter app.  
**TR:** Bu servis Flutter uygulamasi icin kimlik dogrulama, kullanici profili ve vucut metrik gecmisini yonetir.

## ✨ Core Features / Temel Ozellikler

- 🔐 **Auth (JWT):** Register, login, password reset with OTP  
  Kayit, giris ve OTP ile sifre sifirlama
- 🧾 **User Profile API:** Create, list, read, update user profiles  
  Profil olusturma, listeleme, detay ve guncelleme
- 📈 **Metric API:** Save and fetch weight/BMI measurements  
  Kilo/BMI olcumlerini kaydetme ve listeleme
- 🛡️ **App Security:** API key middleware, JWT middleware, security headers  
  API key, JWT ve guvenlik header katmanlari
- ⏱️ **Rate Limiting:** Login and forgot-password throttling  
  Giris ve sifremi unuttum endpointleri icin limit
- 🗃️ **Auto Migrations:** Versioned DB migrations on startup  
  Uygulama acilisinda versiyonlu migration calistirma

## 🧱 Tech Stack / Teknoloji Yigini

- **Go 1.23**
- **MySQL 8.0**
- **gorilla/mux**
- **golang-jwt (HS256)**
- **Resend HTTP API** (email sending / e-posta gonderimi)
- **Docker + docker compose**

## 🗂️ Project Structure / Proje Yapisi

```text
body-metrics-backend/
├── cmd/
│   └── server/
│       └── main.go                 # Entry point / Giris noktasi
├── internal/
│   ├── config/
│   │   └── config.go              # Env config / Ortam degiskenleri
│   ├── db/
│   │   ├── mysql.go               # DB connection pool
│   │   └── migration.go           # Versioned migrations
│   ├── domain/                    # DTO + entities
│   ├── repository/                # SQL layer
│   ├── handler/                   # HTTP handlers
│   ├── middleware/                # Auth, API key, rate limit, CORS, headers
│   └── service/
│       └── email_service.go       # Resend integration
├── Dockerfile
├── docker-compose.yml
├── .env.example
├── go.mod
└── go.sum
```

## 🔌 API Overview / API Ozeti

**Base Path:** `/api/v1`

### Endpoint Matrix / Endpoint Matrisi

| Method | Path | Rate Limit | Auth | EN / TR |
|---|---|---|---|---|
| GET | `/health` | - | - | Health check / Saglik kontrolu |
| POST | `/auth/register` | - | API Key | Register and return JWT / Kayit olup JWT doner |
| POST | `/auth/login` | 5 req / 15 min | API Key | Login and return JWT / Giris yapip JWT doner |
| POST | `/auth/forgot-password` | 3 req / 60 min | API Key | Send OTP mail / OTP e-postasi gonderir |
| POST | `/auth/reset-password` | - | API Key | Reset password by OTP / OTP ile sifre sifirlar |
| POST | `/users` | - | API Key + JWT | Create profile / Profil olusturur |
| GET | `/users` | - | API Key + JWT | List profiles / Profilleri listeler |
| GET | `/users/{id}` | - | API Key + JWT | Get profile detail / Profil detayi |
| PATCH | `/users/{id}` | - | API Key + JWT | Partial profile update / Kismi profil guncelleme |
| POST | `/users/{id}/metrics` | - | API Key + JWT | Add metric / Olcum ekler |
| GET | `/users/{id}/metrics` | - | API Key + JWT | List user metrics / Kullanici olcumleri |

## 🗄️ Database Schema / Veritabani Semasi

### `accounts`
- `id` (PK), `email` (unique), `password_hash`, `created_at`, `updated_at`

### `password_reset_tokens`
- `id` (PK), `account_id` (FK), `token`, `expires_at`, `used`, `created_at`

### `users`
- `id` (PK), `name`, `surname`, `gender`, `avatar`, `height`, `birth_of_date`, `created_at`, `updated_at`

### `user_metrics`
- `id` (PK), `user_id` (FK), `date`, `weight`, `height`, `bmi`, `weight_diff`, `body_metric`, `created_at`

## 🛡️ Security Model / Guvenlik Modeli

### Middleware Chain / Middleware Zinciri

```text
Request -> CORS -> SecurityHeaders -> MaxBytesReader(1MB) -> APIKey -> (JWT for protected routes)
```

### Security Headers / Guvenlik Headerlari

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Strict-Transport-Security: max-age=31536000; includeSubDomains`

### Password Reset Flow / Sifre Sifirlama Akisi

1. `POST /auth/forgot-password` always returns success-style response  
   E-posta var/yok bilgisini ifsa etmez
2. Secure 6-digit OTP is generated and stored with TTL  
   Guvenli 6 haneli OTP uretilir ve sureli kaydedilir
3. OTP email is sent via Resend  
   OTP Resend ile e-posta olarak gonderilir
4. `POST /auth/reset-password` validates token and updates password hash  
   Token dogrulanir ve sifre hash guncellenir

## ⚙️ Setup / Kurulum

### Docker (Recommended / Onerilen)

```bash
cp .env.example .env
# Fill env values / Degerleri doldur
docker compose up -d
```

API: `http://localhost:8080`

### Local Run / Lokal Calistirma

```bash
cp .env.example .env
# MySQL must be running / MySQL calisir olmali
go run ./cmd/server
```

## 🌍 Environment Variables / Ortam Degiskenleri

| Variable | Default | Description (EN / TR) |
|---|---|---|
| `DB_HOST` | `localhost` | MySQL host |
| `DB_PORT` | `3306` | MySQL port |
| `DB_USER` | `bodymetrics` | MySQL user |
| `DB_PASSWORD` | `bodymetrics_pass` | MySQL password |
| `DB_NAME` | `bodymetrics` | Database name |
| `JWT_SECRET` | - | JWT secret (required / zorunlu) |
| `API_KEY` | - | App-level API key (empty disables check / bos ise kontrol kapali) |
| `PORT` | `8080` | API port |
| `RESEND_API_KEY` | - | Resend API key |
| `EMAIL_FROM` | `BodyMetrics <noreply@send.bodymetrics.life>` | Sender identity / Gonderen bilgisi |
| `ALLOWED_ORIGINS` | `*` | CORS allowed origins |

## ☁️ Production Notes / Production Notlari

- **Production Base URL:** `https://api.bodymetrics.life/api/v1`
- **Hosting:** Railway
- **DNS:** Namecheap (CNAME + SPF + DKIM + DMARC)
- **TLS:** Managed by Railway

## 🔄 Migration System / Migration Sistemi

**EN:** Migrations are defined in `internal/db/migration.go` and run automatically at startup in order.  
**TR:** Migrationlar `internal/db/migration.go` icinde tanimlidir ve uygulama acilisinda sirali olarak otomatik calisir.

To add a migration / Yeni migration eklemek icin:
1. Add next version entry (e.g. `004_...`) to migrations list.
2. Keep SQL idempotent when possible.
3. Restart service and verify `schema_migrations`.

## 🧭 next_step

1. **Tenant isolation (critical):** add `account_id` relation to `users`, scope all user/metric queries by token account.
2. **IDOR protection:** enforce ownership checks for `/users/{id}` and `/users/{id}/metrics`.
3. **Password reset replay fix:** mark reset token as used immediately after successful password update.
4. **Rate-limit hardening:** trust `X-Forwarded-For` only behind trusted proxy; otherwise use `RemoteAddr`.
5. **HTTP timeouts:** move to explicit `http.Server` with read/write/idle timeouts.
6. **CORS hardening:** replace wildcard origins in production with strict allow-list.
7. **Validation hardening:** central validator for email/password policy.
8. **Safe logging:** mask PII and avoid sensitive payload logs.
9. **Authorization tests:** add integration tests for cross-account access attempts.
