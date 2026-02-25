# Body Metrics Backend

BodyMetrics Flutter uygulamasının REST API backend servisi. Go ile yazılmış, MySQL veritabanı kullanan, Docker ile deploy edilebilen hafif bir API.

## Proje Amacı

BodyMetrics, kullanıcıların boy, kilo ve BMI gibi sağlık metriklerini takip ettiği bir mobil uygulamadır. Bu backend servisi, kullanıcı ve ölçüm verilerini merkezi bir veritabanında saklar ve Flutter uygulamasına REST API üzerinden sunar.

## Teknolojiler

- **Go 1.23** — API servisi
- **MySQL 8.0** — Veritabanı
- **gorilla/mux** — HTTP router
- **golang-jwt** — JWT authentication (HS256)
- **net/smtp** — E-posta gönderimi (standart kütüphane, sıfır bağımlılık)
- **Docker** — Containerization

## Proje Yapısı

```
body-metrics-backend/
├── cmd/
│   └── server/
│       └── main.go                      # Entry point, router ve DI kurulumu
├── internal/
│   ├── config/
│   │   └── config.go                    # Environment değişkenlerinden config okuma
│   ├── db/
│   │   ├── mysql.go                     # MySQL connection pool
│   │   └── migration.go                 # Otomatik migration sistemi (versiyonlu, transaction'lı)
│   ├── domain/
│   │   ├── user.go                      # User struct
│   │   ├── metric.go                    # UserMetric struct
│   │   ├── auth.go                      # Token request/response struct'ları
│   │   └── password_reset.go            # ForgotPasswordRequest, ResetPasswordRequest, PasswordResetToken
│   ├── repository/
│   │   ├── account_repo.go              # Account CRUD (email + password_hash + UpdatePassword)
│   │   ├── user_repo.go                 # User CRUD
│   │   ├── metric_repo.go               # Metric CRUD
│   │   └── reset_token_repo.go          # Password reset token CRUD
│   ├── service/
│   │   └── email_service.go             # SMTP e-posta gönderimi (Gmail STARTTLS)
│   ├── handler/
│   │   ├── auth_handler.go              # register, login, forgot-password, reset-password
│   │   ├── user_handler.go              # /users endpoint'leri
│   │   ├── metric_handler.go            # /users/:id/metrics endpoint'leri
│   │   └── response.go                  # JSON response helper'ları
│   └── middleware/
│       ├── auth.go                      # JWT doğrulama middleware'i + token üretimi
│       ├── apikey.go                    # API key middleware'i
│       ├── ratelimit.go                 # Sliding window rate limiter (in-memory, sync.Map)
│       └── security.go                  # Security headers + CORS middleware
├── docker-compose.yml                   # MySQL + API + PhpMyAdmin
├── Dockerfile                           # Multi-stage build (golang:1.23-alpine → alpine:3.20)
├── .env.example                         # Örnek environment değişkenleri
├── go.mod
└── go.sum
```

## API Endpoint'leri

| Method | Path | Rate Limit | Auth | Açıklama |
|--------|------|-----------|------|----------|
| `GET` | `/api/v1/health` | — | — | Sağlık kontrolü |
| `POST` | `/api/v1/auth/register` | — | API Key | Hesap oluştur → JWT döner |
| `POST` | `/api/v1/auth/login` | 5 istek / 15 dk | API Key | Giriş yap → JWT döner |
| `POST` | `/api/v1/auth/forgot-password` | 3 istek / 60 dk | API Key | 6 haneli OTP e-posta gönder |
| `POST` | `/api/v1/auth/reset-password` | — | API Key | OTP + yeni şifre ile sıfırla |
| `POST` | `/api/v1/users` | — | JWT | Yeni kullanıcı oluştur |
| `GET` | `/api/v1/users` | — | JWT | Tüm kullanıcıları listele |
| `GET` | `/api/v1/users/:id` | — | JWT | Kullanıcı detayı |
| `PATCH` | `/api/v1/users/:id` | — | JWT | Kullanıcı güncelle |
| `POST` | `/api/v1/users/:id/metrics` | — | JWT | Yeni ölçüm ekle |
| `GET` | `/api/v1/users/:id/metrics` | — | JWT | Tüm ölçümleri getir |

## Veritabanı Şeması

### accounts
| Kolon | Tip | Açıklama |
|-------|-----|----------|
| id | BIGINT (PK) | Auto increment |
| email | VARCHAR(255) | Benzersiz e-posta |
| password_hash | VARCHAR(255) | bcrypt hash |
| created_at | DATETIME | Oluşturma zamanı |
| updated_at | DATETIME | Güncelleme zamanı |

### password_reset_tokens
| Kolon | Tip | Açıklama |
|-------|-----|----------|
| id | BIGINT (PK) | Auto increment |
| account_id | BIGINT (FK) | accounts.id referansı |
| token | VARCHAR(6) | 6 haneli OTP (crypto/rand) |
| expires_at | DATETIME | Son geçerlilik tarihi (15 dk) |
| used | TINYINT(1) | Kullanılmış mı? |
| created_at | DATETIME | Oluşturma zamanı |

### users
| Kolon | Tip | Açıklama |
|-------|-----|----------|
| id | BIGINT (PK) | Auto increment |
| name | VARCHAR(100) | Ad |
| surname | VARCHAR(100) | Soyad |
| gender | TINYINT | 0=erkek, 1=kadın |
| avatar | VARCHAR(50) | Profil resmi (pr1, pr2, ...) |
| height | INT | Boy (cm) |
| birth_of_date | VARCHAR(20) | Doğum tarihi |
| created_at | DATETIME | Oluşturma zamanı |
| updated_at | DATETIME | Güncelleme zamanı |

### user_metrics
| Kolon | Tip | Açıklama |
|-------|-----|----------|
| id | BIGINT (PK) | Auto increment |
| user_id | BIGINT (FK) | users.id referansı |
| date | VARCHAR(20) | Tarih (dd-MM-yyyy, legacy) |
| weight | DOUBLE | Kilo (kg) |
| height | INT | Boy (cm) |
| bmi | DOUBLE | BMI değeri |
| weight_diff | DOUBLE | Önceki ölçümle fark |
| body_metric | VARCHAR(30) | BMI kategorisi |
| created_at | VARCHAR(30) | ISO8601 canonical timestamp |

## Güvenlik

### Middleware Zinciri
```
Request → CORSMiddleware → SecurityHeaders → MaxBytesReader(1MB) → APIKeyMiddleware → ...
```

### Security Headers (her response'a eklenir)
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Strict-Transport-Security: max-age=31536000; includeSubDomains`

### Rate Limiting
- Login: IP başına **5 istek / 15 dakika**
- Forgot Password: IP başına **3 istek / 60 dakika**
- In-memory sliding window, `sync.Map` tabanlı, sıfır bağımlılık

### Şifremi Unuttum Akışı
1. `POST /auth/forgot-password` → account bulunsa da bulunmasa da `200 OK` döner (e-posta enumeration koruması)
2. Arka planda `crypto/rand` ile 6 haneli OTP üretilir, DB'ye yazılır (15 dk TTL)
3. Gmail SMTP/STARTTLS üzerinden OTP gönderilir
4. `POST /auth/reset-password` → OTP doğrulanır → bcrypt hash → şifre güncellenir → token kullanıldı işaretlenir

## Kurulum

### Docker ile (Önerilen)

```bash
cp .env.example .env
# .env dosyasındaki değerleri doldur (JWT_SECRET, SMTP_* vb.)

docker compose up -d
```

API `http://localhost:8080` adresinde çalışmaya başlar.

### Manuel

```bash
# MySQL çalışıyor olmalı
cp .env.example .env
# .env dosyasını düzenle

go run ./cmd/server
```

## Environment Değişkenleri

| Değişken | Varsayılan | Açıklama |
|----------|-----------|----------|
| `DB_HOST` | `localhost` | MySQL host |
| `DB_PORT` | `3306` | MySQL port |
| `DB_USER` | `bodymetrics` | MySQL kullanıcı |
| `DB_PASSWORD` | `bodymetrics_pass` | MySQL şifre |
| `DB_NAME` | `bodymetrics` | Veritabanı adı |
| `JWT_SECRET` | — | JWT imzalama anahtarı **(zorunlu)** |
| `API_KEY` | — | API key (boş = devre dışı) |
| `PORT` | `8080` | Sunucu portu |
| `SMTP_HOST` | `smtp.gmail.com` | SMTP sunucu |
| `SMTP_PORT` | `587` | SMTP port (STARTTLS) |
| `SMTP_USER` | — | Gmail adresi |
| `SMTP_PASS` | — | Gmail App Password |
| `SMTP_FROM` | — | Gönderen adı ve adresi |
| `ALLOWED_ORIGINS` | `*` | CORS izin verilen origin'ler |

## Railway Deployment

Railway dashboard → proje → **Variables** sekmesine aşağıdakileri ekle:

```
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-gmail-app-password
SMTP_FROM=BodyMetrics <your-email@gmail.com>
ALLOWED_ORIGINS=*
```

Deploy tetiklendiğinde `003_create_password_reset_tokens` migration'ı otomatik çalışır.

> **Gmail App Password:** Google Account → Security → 2-Step Verification → App Passwords

## Migration Sistemi

Migration'lar `internal/db/migration.go` dosyasındaki `migrations` slice'ında tanımlıdır. Sunucu her başladığında uygulanmamış migration'ları sırayla transaction içinde çalıştırır. Hata olursa rollback yapılır ve sunucu durur.

Yeni migration eklemek için `004_...` versiyonlu yeni bir struct eklemek yeterlidir.
