package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/kajve/ws-gateway/src/domain/entities"
	"github.com/kajve/ws-gateway/src/core"
)

// LoteRepository implementa domain.LoteRepository sobre Postgres.
type LoteRepository struct {
	db *core.DB
}

func NewLoteRepository(db *core.DB) *LoteRepository {
	return &LoteRepository{db: db}
}

// GetByID trae el id_usuario dueño del lote, para verificar propiedad ANTES
// de abrir el WebSocket (el usuario todavía no está "confirmado" en este
// punto, así que no aplica RLS aquí — mismo criterio que
// api-mobile.LoteRepository.GetByID, que tampoco usa BeginTx para esta
// consulta puntual).
func (r *LoteRepository) GetByID(ctx context.Context, loteID int) (*entities.Lote, error) {
	l := &entities.Lote{}
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id_lote, id_usuario FROM lotes_cafe WHERE id_lote = $1
	`, loteID).Scan(&l.ID, &l.UsuarioID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("lote_repository: error consultando lote: %w", err)
	}
	return l, nil
}
