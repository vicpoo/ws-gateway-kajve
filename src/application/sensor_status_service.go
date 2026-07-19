package application

import (
	"context"
	"time"

	"github.com/kajve/ws-gateway/src/domain"
	"github.com/kajve/ws-gateway/src/domain/entities"
)

// SensorStatusService decide si un sensor está "conectado" comparando la
// marca de tiempo de su última lectura contra un umbral. El ESP32 publica
// cada 5s (ver firmware), así que un umbral de unos pocos ciclos de envío
// perdidos es suficiente para no marcar "desconectado" por un solo mensaje
// tardío o perdido en tránsito.
type SensorStatusService struct {
	repo      domain.SensorStatusRepository
	threshold time.Duration
}

func NewSensorStatusService(repo domain.SensorStatusRepository, threshold time.Duration) *SensorStatusService {
	return &SensorStatusService{repo: repo, threshold: threshold}
}

// Estado calcula el estado de conexión de cada sensor del usuario.
func (s *SensorStatusService) Estado(ctx context.Context, usuarioID int) ([]entities.EstadoSensor, error) {
	estados, err := s.repo.GetUltimasLecturas(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	ahora := time.Now()
	for i := range estados {
		ultima := estados[i].UltimaLectura
		estados[i].Conectado = ultima != nil && ahora.Sub(*ultima) <= s.threshold
	}
	return estados, nil
}
