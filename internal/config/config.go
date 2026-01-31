package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPass     string
	DBName     string
	ServerPort string
	JWTSecret  string
}

func Load() *Config {
	_ = godotenv.Load() // 加载项目根目录下的 .env

	return &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPass:     getEnv("DB_PASSWORD", "123456"),
		DBName:     getEnv("DB_NAME", "iot"),
		ServerPort: getEnv("SERVER_PORT", "8888"),
		JWTSecret:  getEnv("JWT_SECRET", "wlw4"),
	}
}
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
func (c *Config) GetDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser, c.DBPass, c.DBHost, c.DBPort, c.DBName)
}
