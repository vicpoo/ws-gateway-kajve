//src/infrastructure/dependencies/config.go
package dependencies

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config son las variables de entorno del servicio.
type Config struct {
	Port                   string
	DatabaseURL            string
	RedisAddr              string
	RedisPassword          string
	RedisDB                int
	JWTSecret              string
	AllowedOrigin          string
	HistoryLimit           int
	SensorOfflineThreshold time.Duration
}

// LoadConfig lee y valida la configuración mínima para arrancar.
func LoadConfig() (*Config, error) {
	historyLimit, err := strconv.Atoi(getEnv("HISTORY_LIMIT", "100"))
	if err != nil {
		return nil, fmt.Errorf("config: HISTORY_LIMIT inválido: %w", err)
	}

	// SENSOR_OFFLINE_THRESHOLD_SECONDS: cuánto tiempo sin una lectura nueva
	// antes de marcar un sensor como "desconectado". El ESP32 manda datos
	// cada 5s (firmware actual) — 20s da margen a ~4 ciclos de envío
	// perdidos antes de marcarlo desconectado, evitando falsos negativos
	// por un solo mensaje tardío o un reconnect de MQTT.
	offlineThresholdSeconds, err := strconv.Atoi(getEnv("SENSOR_OFFLINE_THRESHOLD_SECONDS", "20"))
	if err != nil {
		return nil, fmt.Errorf("config: SENSOR_OFFLINE_THRESHOLD_SECONDS inválido: %w", err)
	}

	cfg := &Config{
		Port:                   getEnv("PORT", "8002"),
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		RedisAddr:              getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:          os.Getenv("REDIS_PASSWORD"),
		RedisDB:                0,
		JWTSecret:              os.Getenv("JWT_SECRET"),
		AllowedOrigin:          getEnv("WS_ALLOWED_ORIGIN", "*"),
		HistoryLimit:           historyLimit,
		SensorOfflineThreshold: time.Duration(offlineThresholdSeconds) * time.Second,
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: falta la variable de entorno DATABASE_URL")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("config: falta la variable de entorno JWT_SECRET (debe ser igual a la que usa api-mobile)")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
