package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	AppName      string
	AppEnv       string
	Port         string
	LogLevel     string
	DBDriver     string
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
	DBSSLMode    string
	JWTSecret    string
	MasterAPIKey string
}

func LoadConfig() *Config {
	loadDotEnv(".env")

	return &Config{
		AppName:      getEnv("APP_NAME", "be-remote-device"),
		AppEnv:       getEnv("APP_ENV", "development"),
		Port:         getEnv("APP_PORT", "8080"),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		DBDriver:     getEnv("DB_DRIVER", "postgres"),
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "postgres"),
		DBPassword:   getEnv("DB_PASSWORD", "postgres"),
		DBName:       getEnv("DB_NAME", "remote_device_db"),
		DBSSLMode:    getEnv("DB_SSLMODE", "disable"),
		JWTSecret:    getEnv("JWT_SECRET", "super-secret-remote-device-key-change-me"),
		MasterAPIKey: getEnv("MASTER_API_KEY", "dev-master-key-12345"),
	}
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func loadDotEnv(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			if _, exists := os.LookupEnv(k); !exists {
				_ = os.Setenv(k, v)
			}
		}
	}
}
