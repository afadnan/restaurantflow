package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv   string
	HTTPPort int

	Database DatabaseConfig
	Redis    RedisConfig
	Storage  StorageConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

type RedisConfig struct {
	Host string
	Port int
}

type StorageConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
}

func Load() (Config, error) {
	if err := loadLocalEnv(); err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppEnv: getEnv("APP_ENV", "local"),
	}

	var err error

	cfg.HTTPPort, err = getRequiredInt("HTTP_PORT")
	if err != nil {
		return Config{}, err
	}

	cfg.Database, err = loadDatabaseConfig()
	if err != nil {
		return Config{}, err
	}

	cfg.Redis, err = loadRedisConfig()
	if err != nil {
		return Config{}, err
	}

	cfg.Storage = loadStorageConfig()

	return cfg, nil
}

// loadLocalEnv loads the repository .env file for local development.
//
// Existing environment variables always win over values in .env.
// Production/staging environments should provide configuration through
// the process environment or their secret/configuration manager.
func loadLocalEnv() error {
	appEnv := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))

	if appEnv != "" &&
		appEnv != "local" &&
		appEnv != "development" &&
		appEnv != "dev" {
		return nil
	}

	envPath, err := findEnvFile()
	if err != nil {
		return err
	}

	if envPath == "" {
		return nil
	}

	if err := godotenv.Load(envPath); err != nil {
		return fmt.Errorf("load environment file %q: %w", envPath, err)
	}

	return nil
}

func findEnvFile() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	for {
		candidate := filepath.Join(dir, ".env")

		info, err := os.Stat(candidate)
		if err == nil {
			if info.IsDir() {
				return "", fmt.Errorf("%s is a directory", candidate)
			}

			return candidate, nil
		}

		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect environment file %q: %w", candidate, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}

		dir = parent
	}
}

func loadDatabaseConfig() (DatabaseConfig, error) {
	port, err := getRequiredInt("DATABASE_PORT")
	if err != nil {
		return DatabaseConfig{}, err
	}

	cfg := DatabaseConfig{
		Host:     os.Getenv("DATABASE_HOST"),
		Port:     port,
		Name:     os.Getenv("DATABASE_NAME"),
		User:     os.Getenv("DATABASE_USER"),
		Password: os.Getenv("DATABASE_PASSWORD"),
		SSLMode:  getEnv("DATABASE_SSLMODE", "disable"),
	}

	if cfg.Host == "" {
		return DatabaseConfig{}, errors.New("DATABASE_HOST is required")
	}

	if cfg.Name == "" {
		return DatabaseConfig{}, errors.New("DATABASE_NAME is required")
	}

	if cfg.User == "" {
		return DatabaseConfig{}, errors.New("DATABASE_USER is required")
	}

	if cfg.Password == "" {
		return DatabaseConfig{}, errors.New("DATABASE_PASSWORD is required")
	}

	return cfg, nil
}

func loadRedisConfig() (RedisConfig, error) {
	port, err := getRequiredInt("REDIS_PORT")
	if err != nil {
		return RedisConfig{}, err
	}

	cfg := RedisConfig{
		Host: os.Getenv("REDIS_HOST"),
		Port: port,
	}

	if cfg.Host == "" {
		return RedisConfig{}, errors.New("REDIS_HOST is required")
	}

	return cfg, nil
}

func loadStorageConfig() StorageConfig {
	return StorageConfig{
		Endpoint:  os.Getenv("STORAGE_ENDPOINT"),
		AccessKey: os.Getenv("STORAGE_ACCESS_KEY"),
		SecretKey: os.Getenv("STORAGE_SECRET_KEY"),
		Bucket:    os.Getenv("STORAGE_BUCKET"),
	}
}

func getEnv(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	return value
}

func getRequiredInt(key string) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return 0, fmt.Errorf("%s is required", key)
	}

	result, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}

	if result <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", key)
	}

	return result, nil
}
