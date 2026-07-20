//src/infrastructure/repository/lectura_repository.go
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/kajve/ws-gateway/src/core"
	"github.com/kajve/ws-gateway/src/domain/entities"
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
//
// Columnas alineadas con la migración de BD: humedad y la vieja columna
// lluvia ya no existen; se leen en su lugar lluvia_analog y
// lluvia_detectada.
func (r *LecturaRepository) GetUltimas(ctx context.Context, loteID int, usuarioID int, limit int) ([]entities.PuntoHistorico, error) {
	var puntos []entities.PuntoHistorico
	err := r.db.WithUserContext(ctx, usuarioID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id_sensor, temperatura, temperatura_grano, luz,
			       lluvia_analog, lluvia_detectada, humedad_grano,
			       presion_hpa, altitud_m, "timestamp"
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
				&p.SensorID, &p.Lectura.Temperatura, &p.Lectura.TemperaturaGrano, &p.Lectura.Luz,
				&p.Lectura.LluviaAnalog, &p.Lectura.LluviaDetectada, &p.Lectura.HumedadGrano,
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

// GetResumen calcula agregados sobre TODAS las lecturas del lote (no solo
// las últimas N, a diferencia de GetUltimas) — es lo que alimenta
// GET /lotes/{id}/resumen para la vista de Monitoreo. Si el lote no tiene
// lecturas todavía, COUNT sale en 0 y el resto de las columnas agregadas
// salen en NULL (comportamiento estándar de SQL: MIN/AVG/MAX sobre cero
// filas es NULL), lo cual mapea directo a punteros nil sin necesitar
// manejo especial.
func (r *LecturaRepository) GetResumen(ctx context.Context, loteID int, usuarioID int) (*entities.ResumenLote, error) {
	var resumen entities.ResumenLote
	err := r.db.WithUserContext(ctx, usuarioID, func(tx pgx.Tx) error {
		// Los MIN/MAX/AVG se castean explícitamente a float8: Postgres
		// devuelve el tipo original de la columna en MIN/MAX (humedad_grano
		// es smallint) y "numeric" en AVG de una columna entera — el cast
		// evita cualquier ambigüedad al escanear directo a *float64 en Go.
		row := tx.QueryRow(ctx, `
			SELECT
				COUNT(*),
				MIN("timestamp"), MAX("timestamp"),
				MIN(temperatura)::float8, AVG(temperatura)::float8, MAX(temperatura)::float8,
				MIN(temperatura_grano)::float8, AVG(temperatura_grano)::float8, MAX(temperatura_grano)::float8,
				MIN(humedad_grano)::float8, AVG(humedad_grano)::float8, MAX(humedad_grano)::float8,
				MIN(presion_hpa)::float8, AVG(presion_hpa)::float8, MAX(presion_hpa)::float8
			FROM lecturas_ambientales
			WHERE id_lote = $1
		`, loteID)
		return row.Scan(
			&resumen.TotalLecturas,
			&resumen.PrimeraLectura, &resumen.UltimaLectura,
			&resumen.TemperaturaMin, &resumen.TemperaturaProm, &resumen.TemperaturaMax,
			&resumen.TemperaturaGranoMin, &resumen.TemperaturaGranoProm, &resumen.TemperaturaGranoMax,
			&resumen.HumedadGranoMin, &resumen.HumedadGranoProm, &resumen.HumedadGranoMax,
			&resumen.PresionHpaMin, &resumen.PresionHpaProm, &resumen.PresionHpaMax,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("lectura_repository: error calculando resumen: %w", err)
	}
	return &resumen, nil
}