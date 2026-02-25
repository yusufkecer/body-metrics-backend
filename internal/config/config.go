package config

import "os"

type Config struct {
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	JWTSecret      string
	APIKey         string
	Port           string
	SMTPHost       string
	SMTPPort       string
	SMTPUser       string
	SMTPPass       string
	SMTPFrom       string
	AllowedOrigins string
}

func Load() *Config {
	return &Config{
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "3306"),
		DBUser:         getEnv("DB_USER", "bodymetrics"),
		DBPassword:     getEnv("DB_PASSWORD", "bodymetrics_pass"),
		DBName:         getEnv("DB_NAME", "bodymetrics"),
		JWTSecret:      getEnv("JWT_SECRET", ""),
		APIKey:         getEnv("API_KEY", ""),
		Port:           getEnv("PORT", "8080"),
		SMTPHost:       getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:       getEnv("SMTP_PORT", "587"),
		SMTPUser:       getEnv("SMTP_USER", ""),
		SMTPPass:       getEnv("SMTP_PASS", ""),
		SMTPFrom:       getEnv("SMTP_FROM", ""),
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "*"),
	}
}

func (c *Config) DSN() string {
	return c.DBUser + ":" + c.DBPassword + "@tcp(" + c.DBHost + ":" + c.DBPort + ")/" + c.DBName + "?parseTime=true&charset=utf8mb4"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
