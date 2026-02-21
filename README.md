# Body Metrics Backend

BodyMetrics Flutter uygulamasının REST API backend servisi. Go ile yazılmış, MySQL veritabanı kullanan, Docker ile deploy edilebilen hafif bir API.

## Proje Amacı

BodyMetrics, kullanıcıların boy, kilo ve BMI gibi sağlık metriklerini takip ettiği bir mobil uygulamadır. Bu backend servisi, kullanıcı ve ölçüm verilerini merkezi bir veritabanında saklar ve Flutter uygulamasına REST API üzerinden sunar.

## Teknolojiler

- **Go 1.23** — API servisi
- **MySQL 8.0** — Veritabanı
- **gorilla/mux** — HTTP router
- **golang-jwt** — JWT authentication (HS256)
- **Docker** — Containerization

## Proje Yapısı

```
body-metrics-backend/
├── cmd/
│   └── server/
│       └── main.go                 # Entry point, router ve DI kurulumu
├── internal/
│   ├── config/
│   │   └── config.go              # Environment değişkenlerinden config okuma
│   ├── db/
│   │   ├── mysql.go               # MySQL connection pool
│   │   └── migration.go           # Otomatik migration sistemi (versiyonlu, transaction'lı)
│   ├── domain/
│   │   ├── user.go                # User struct
│   │   ├── metric.go              # UserMetric struct
│   │   └── auth.go                # Token request/response struct'ları
│   ├── repository/
│   │   ├── user_repo.go           # User CRUD (Create, GetByID, GetAll, Update)
│   │   └── metric_repo.go         # Metric CRUD (Create, GetByUserID)
│   ├── handler/
│   │   ├── auth_handler.go        # POST /auth/register, POST /auth/login
│   │   ├── user_handler.go        # /users endpoint'leri
│   │   ├── metric_handler.go      # /users/:id/metrics endpoint'leri
│   │   └── response.go            # JSON response helper'ları
│   └── middleware/
│       └── auth.go                # JWT doğrulama middleware'i
├── docker-compose.yml              # MySQL + API servisi
├── Dockerfile                      # Multi-stage build (golang:1.23-alpine → alpine:3.20)
├── .env.example                    # Örnek environment değişkenleri
├── go.mod
└── go.sum
```

## API Endpoint'leri

| Method | Path | Auth | Açıklama |
|--------|------|------|----------|
| `POST` | `/api/v1/auth/register` | - | E-posta ve şifre ile hesap oluştur |
| `POST` | `/api/v1/auth/login` | - | E-posta ve şifre ile giriş yap |
| `POST` | `/api/v1/users` | JWT | Yeni kullanıcı oluştur |
| `GET` | `/api/v1/users` | JWT | Tüm kullanıcıları listele |
| `GET` | `/api/v1/users/:id` | JWT | Kullanıcı detayı |
| `PATCH` | `/api/v1/users/:id` | JWT | Kullanıcı güncelle (isim, boy, cinsiyet vb.) |
| `POST` | `/api/v1/users/:id/metrics` | JWT | Yeni ölçüm ekle (kilo, BMI, vb.) |
| `GET` | `/api/v1/users/:id/metrics` | JWT | Kullanıcının tüm ölçümlerini getir |

## Veritabanı Şeması

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
| date | VARCHAR(20) | Tarih (dd-MM-yyyy) |
| weight | DOUBLE | Kilo (kg) |
| height | INT | Boy (cm) |
| bmi | DOUBLE | BMI değeri |
| weight_diff | DOUBLE | Önceki ölçümle fark |
| body_metric | VARCHAR(30) | BMI kategorisi (underweight, normal, vb.) |
| created_at | VARCHAR(30) | ISO8601 timestamp |

## Kurulum

### Docker ile (Önerilen)

```bash
cp .env.example .env
# .env dosyasındaki JWT_SECRET değerini değiştir

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

## Auth Akışı

1. Flutter uygulaması `POST /auth/register` ile hesap açar veya `POST /auth/login` ile giriş yapar
2. API JWT token döner
3. Sonraki tüm isteklerde `Authorization: Bearer <token>` header'ı gönderilir
4. Token süresi: 30 gün

## Migration Sistemi

Migration'lar `internal/db/migration.go` dosyasında tanımlıdır. Sunucu her başladığında:

1. `schema_migrations` tablosunu kontrol eder
2. Uygulanmamış migration'ları sırayla çalıştırır (transaction içinde)
3. Her başarılı migration loglanır
4. Hata olursa rollback yapılır ve sunucu durur

Yeni migration eklemek için `migration.go` dosyasındaki `migrations` slice'ına yeni bir struct eklemek yeterlidir.
