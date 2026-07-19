// Package redis se suscribe al canal de tiempo real que publica
// ingesta-iot (ver infrastructure/redis/publisher.go allá).
package redis

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

// Subscriber envuelve go-redis.
type Subscriber struct {
	client *goredis.Client
}

func NewSubscriber(addr, password string, db int) *Subscriber {
	return &Subscriber{
		client: goredis.NewClient(&goredis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
	}
}

// Ping verifica la conexión. No es fatal si falla (ver container.go): el
// Gateway sigue sirviendo histórico aunque Redis no esté disponible.
func (s *Subscriber) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// Subscribe se suscribe al canal user:<usuarioID> — el mismo canal donde
// ingesta-iot publica cada lectura nueva — y devuelve un canal con los
// payloads crudos (ya en el JSON que publica Ingesta, se reenvían tal
// cual) más una función para cancelar la suscripción.
func (s *Subscriber) Subscribe(ctx context.Context, usuarioID int) (<-chan []byte, func(), error) {
	channel := fmt.Sprintf("user:%d", usuarioID)
	pubsub := s.client.Subscribe(ctx, channel)

	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return nil, nil, fmt.Errorf("redis: error suscribiéndose a %q: %w", channel, err)
	}

	out := make(chan []byte)
	go func() {
		defer close(out)
		msgs := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				select {
				case out <- []byte(msg.Payload):
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	unsubscribe := func() {
		_ = pubsub.Close()
	}
	return out, unsubscribe, nil
}

func (s *Subscriber) Close() error {
	return s.client.Close()
}
