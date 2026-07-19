package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/kajve/ws-gateway/src/domain/entities"
	"github.com/kajve/ws-gateway/src/core"
)

// LecturaRepository implementa domain.LecturaRepository sobre Postgres.
type LecturaRepository struct {
	db *core.DB
}

func NewLecturaRepository(db *core.DB) *LecturaRepository {
	return &LecturaRepository{db: db}
}

// GetUltimas trae las últimas `limit` lecturas de un lote, en orden
// cronológico (de más vieja a más nueva) para que el cliente pueda dibujar
// la gráfica directamente sin tener que voltear el arreglo. Corre dentro de
// una transacción con app.current_user_id fijado, respetando la política
// RLS lecturas_por_usuario.
func (r *LecturaRepository) GetUltimas(ctx context.Context, loteID int, usuarioID int, limit int) ([]entities.PuntoHistorico, error) {
	var puntos []entities.PuntoHistorico

	err := r.db.WithUserContext(ctx, usuarioID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id_sensor, temperatura, humedad, temperatura_grano, luz, lluvia,
			       humedad_grano, presion_hpa, altitud_m, "timestamp"
			FROM lecturas_ambientales
			WHERE id_lote = $1
			ORDER BY "timestamp" DESC
			LIMIT $2
		`, loteID, limit)
		if err != nil {
			return fmt.Errorf("lectura_repository: error consultando histórico: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var p entities.PuntoHistorico
			if err := rows.Scan(
				&p.SensorID, &p.Lectura.Temperatura, &p.Lectura.Humedad, &p.Lectura.TemperaturaGrano,
				&p.Lectura.Luz, &p.Lectura.Lluvia, &p.Lectura.HumedadGrano,
				&p.Lectura.PresionHpa, &p.Lectura.AltitudM, &p.Timestamp,
			); err != nil {
				return fmt.Errorf("lectura_repository: error leyendo fila: %w", err)
			}
			puntos = append(puntos, p)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	// La query trae lo más nuevo primero; invertir a orden cronológico.
	for i, j := 0, len(puntos)-1; i < j; i, j = i+1, j-1 {
		puntos[i], puntos[j] = puntos[j], puntos[i]
	}
	return puntos, nil
}
