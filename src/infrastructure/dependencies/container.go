// Package dependencies arma (wiring) todas las piezas del Gateway.
package dependencies

import (
	"context"
	"fmt"
	"log"

	"github.com/kajve/ws-gateway/src/application"
	"github.com/kajve/ws-gateway/src/infrastructure/auth"
	"github.com/kajve/ws-gateway/src/infrastructure/httpapi"
	"github.com/kajve/ws-gateway/src/infrastructure/redis"
	"github.com/kajve/ws-gateway/src/infrastructure/repository"
	wsinfra "github.com/kajve/ws-gateway/src/infrastructure/websocket"
	"github.com/kajve/ws-gateway/src/core"
)

type Container struct {
	Config              *Config
	DB                  *core.DB
	Redis               *redis.Subscriber
	WSHandler           *wsinfra.Handler
	SensorStatusHandler *httpapi.SensorStatusHandler
}

func NewContainer(ctx context.Context) (*Container, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	db, err := core.NewDB(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("dependencies: %w", err)
	}

	sub := redis.NewSubscriber(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err := sub.Ping(ctx); err != nil {
		log.Printf("dependencies: Redis no disponible (%v) — el Gateway arranca igual, sirviendo histórico; los datos en vivo llegarán en cuanto Redis esté disponible", err)
	} else {
		log.Println("dependencies: conectado a Redis")
	}

	loteRepo := repository.NewLoteRepository(db)
	lecturaRepo := repository.NewLecturaRepository(db)
	sensorStatusRepo := repository.NewSensorStatusRepository(db)
	validator := auth.NewValidator(cfg.JWTSecret)

	gatewayService := application.NewGatewayService(loteRepo, lecturaRepo, sub, validator, cfg.HistoryLimit)
	sensorStatusService := application.NewSensorStatusService(sensorStatusRepo, cfg.SensorOfflineThreshold)

	wsHandler := wsinfra.NewHandler(gatewayService, cfg.AllowedOrigin)
	sensorStatusHandler := httpapi.NewSensorStatusHandler(gatewayService, sensorStatusService)

	return &Container{
		Config:              cfg,
		DB:                  db,
		Redis:               sub,
		WSHandler:           wsHandler,
		SensorStatusHandler: sensorStatusHandler,
	}, nil
}

func (c *Container) Close() {
	if c.Redis != nil {
		_ = c.Redis.Close()
	}
	if c.DB != nil {
		c.DB.Close()
	}
}