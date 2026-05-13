package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

type Config struct {
	DatabaseURL      string
	Port             string
	Env              string
	BusinessTimezone string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file")
	}

	return &Config{
		DatabaseURL:      getEnv("DATABASE_URL", ""),
		Port:             getEnv("PORT", "8080"),
		Env:              getEnv("ENV", "development"),
		BusinessTimezone: getEnv("BUSINESS_TIMEZONE", "Asia/Almaty"),
	}, nil
}

func getEnv(key string, fallback string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}
	return fallback
}

func (c *Config) LogConfig(logger *zap.SugaredLogger) {
	logger.Infow(
		"config loaded",
		"port", c.Port,
		"env", c.Env,
		"BusinessTimezone", c.BusinessTimezone,
	)
}
