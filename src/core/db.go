//src/core/db.go
// Package core contiene la conexión a Postgres, compartido por los
// repositorios de infrastructure. Mismo patrón que ingesta-iot/src/core.
package core

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB envuelve el pool de conexiones de pgx.
type DB struct {
	Pool *pgxpool.Pool
}

// NewDB crea el pool y verifica la conexión de inmediato (fail-fast).
func NewDB(ctx context.Context, connString string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("core: error parseando la cadena de conexión: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("core: error creando el pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("core: error conectando a Postgres: %w", err)
	}
	return &DB{Pool: pool}, nil
}

// WithUserContext ejecuta fn dentro de una transacción con
// app.current_user_id fijado vía set_config (respeta RLS) — mismo patrón
// que ingesta-iot/src/core/db.go, usando set_config en vez del SET con
// interpolación de string que usa api-mobile, porque set_config sí acepta
// placeholders parametrizados ($1).
func (db *DB) WithUserContext(ctx context.Context, usuarioID int, fn func(tx pgx.Tx) error) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("core: error iniciando transacción: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_user_id', $1, true)", strconv.Itoa(usuarioID)); err != nil {
		return fmt.Errorf("core: error fijando app.current_user_id: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Close cierra el pool.
func (db *DB) Close() {
	db.Pool.Close()
}
