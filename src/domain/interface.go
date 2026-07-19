// Package domain define los puertos (interfaces) que implementa
// infrastructure, siguiendo el mismo patrón hexagonal que ingesta-iot.
package domain

import (
	"context"

	"github.com/kajve/ws-gateway/src/domain/entities"
)

// LoteRepository resuelve la propiedad de un lote, para no abrir un
// WebSocket hacia los datos de otro usuario.
type LoteRepository interface {
	GetByID(ctx context.Context, loteID int) (*entities.Lote, error)
}

// LecturaRepository trae el histórico reciente de un lote, para la carga
// inicial de la gráfica al conectar.
type LecturaRepository interface {
	GetUltimas(ctx context.Context, loteID int, usuarioID int, limit int) ([]entities.PuntoHistorico, error)
}

// EventSubscriber se suscribe al canal de tiempo real de un usuario (Redis
// Pub/Sub, canal user:<id>, el mismo que publica ingesta-iot). Devuelve un
// canal de solo lectura con los payloads crudos y una función para cancelar
// la suscripción.
type EventSubscriber interface {
	Subscribe(ctx context.Context, usuarioID int) (<-chan []byte, func(), error)
}
