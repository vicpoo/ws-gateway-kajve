//src/infrastructure/repository/sensor_status_repository.go
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/kajve/ws-gateway/src/core"
	"github.com/kajve/ws-gateway/src/domain/entities"
)

// SensorStatusRepository implementa domain.SensorStatusRepository sobre
// Postgres. Solo trae datos crudos (última lectura por sensor) — la regla
// de "conectado/desconectado" vive en application.SensorStatusService.
type SensorStatusRepository struct {
	db *core.DB
}

func NewSensorStatusRepository(db *core.DB) *SensorStatusRepository {
	return &SensorStatusRepository{db: db}
}

// GetUltimasLecturas trae TODOS los sensores ligados a CUALQUIER lote del
// usuario (sin filtrar por estado — antes solo mostraba lotes
// 'en_proceso', lo que ocultaba sensores de lotes ya cerrados/históricos),
// el id/mac del sensor y la marca de tiempo de su lectura más reciente en
// lecturas_ambientales (NULL si el sensor nunca ha mandado nada).
//
// Un mismo sensor puede haber pasado por más de un lote a lo largo del
// tiempo (placeholder -> lote real, o entre cosechas), así que se usa
// DISTINCT ON para devolver una sola fila por sensor: la de su lote más
// reciente (id_lote más alto).
//
// Corre dentro de una transacción con app.current_user_id fijado,
// respetando la política RLS de lotes_cafe — mismo patrón que
// LecturaRepository.GetUltimas.
func (r *SensorStatusRepository) GetUltimasLecturas(ctx context.Context, usuarioID int) ([]entities.EstadoSensor, error) {
	var estados []entities.EstadoSensor

	err := r.db.WithUserContext(ctx, usuarioID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT DISTINCT ON (s.id_sensor)
			       s.id_sensor, s.mac_address, s.tipo, l.id_lote, l.nombre_lote, u.ultima
			FROM lotes_cafe l
			JOIN sensores s ON s.id_sensor = l.id_sensor
			LEFT JOIN LATERAL (
				SELECT MAX(la."timestamp") AS ultima
				FROM lecturas_ambientales la
				WHERE la.id_sensor = s.id_sensor
			) u ON true
			WHERE l.id_usuario = $1
			ORDER BY s.id_sensor, l.id_lote DESC
		`, usuarioID)
		if err != nil {
			return fmt.Errorf("sensor_status_repository: error consultando estado de sensores: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var e entities.EstadoSensor
			if err := rows.Scan(&e.SensorID, &e.MacAddress, &e.Tipo, &e.LoteID, &e.NombreLote, &e.UltimaLectura); err != nil {
				return fmt.Errorf("sensor_status_repository: error leyendo fila: %w", err)
			}
			estados = append(estados, e)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return estados, nil
}
