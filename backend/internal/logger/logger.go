package logger

import (
	"attendance/config"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Setup(cfg *config.Config) (*zap.SugaredLogger, error) {
	env := strings.ToLower(strings.TrimSpace(cfg.Env))

	var (
		base *zap.Logger
		err  error
	)

	switch env {
	case "dev", "development", "local":
		zapCfg := zap.NewDevelopmentConfig()
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		base, err = zapCfg.Build()
	case "prod", "production":
		zapCfg := zap.NewProductionConfig()
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		base, err = zapCfg.Build()
	default:
		zapCfg := zap.NewProductionConfig()
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		base, err = zapCfg.Build()
	}

	if err != nil {
		return nil, err
	}

	return base.Sugar(), nil
}
