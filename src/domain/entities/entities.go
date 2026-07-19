// Package entities contiene los modelos de dominio del WebSocket Gateway.
package entities

import "time"

// LecturaDTO es la forma de una lectura (histórica o en vivo) que viaja
// hacia el cliente WebSocket. Debe coincidir EXACTAMENTE con
// ingesta-iot/domain/entities.LecturaAmbientalDTO, porque las lecturas en
// vivo llegan tal cual desde Redis en ese formato — este Gateway solo las
// reenvía, no las reconstruye.
type LecturaDTO struct {
	Temperatura      *float64 `json:"temperatura,omitempty"`
	Humedad          *float64 `json:"humedad,omitempty"`
	TemperaturaGrano *float64 `json:"temperatura_grano,omitempty"`
	Luz              *float64 `json:"luz,omitempty"`
	Lluvia           *float64 `json:"lluvia,omitempty"`
	HumedadGrano     *float64 `json:"humedad_grano,omitempty"`
	PresionHpa       *float64 `json:"presion_hpa,omitempty"`
	AltitudM         *float64 `json:"altitud_m,omitempty"`
}

// PuntoHistorico es una lectura ya guardada en Postgres, para la carga
// inicial de la gráfica al conectar (antes de que lleguen datos en vivo).
type PuntoHistorico struct {
	SensorID  int        `json:"sensor_id"`
	Timestamp time.Time  `json:"timestamp"`
	Lectura   LecturaDTO `json:"lectura"`
}

// HistorialMessage es el primer mensaje que manda el Gateway al conectar:
// el respaldo histórico para que la gráfica no arranque vacía. El cliente
// distingue los mensajes por el campo "tipo" ("historial" vs
// "osil.data.updated", este último reenviado tal cual desde Ingesta).
type HistorialMessage struct {
	Tipo   string           `json:"tipo"`
	LoteID int              `json:"lote_id"`
	Puntos []PuntoHistorico `json:"puntos"`
}

// Lote es lo mínimo necesario para verificar propiedad antes de abrir el WS.
type Lote struct {
	ID        int
	UsuarioID int
}

// EstadoSensor resume si un sensor del usuario está "conectado" (mandó una
// lectura hace poco) o "desconectado" (no ha mandado nada en el umbral
// configurado). No depende de sensores.ultima_conexion (esa columna no se
// actualiza en cada mensaje) — se deriva directamente de la lectura más
// reciente en lecturas_ambientales, que es la fuente de verdad real.
type EstadoSensor struct {
	SensorID      int        `json:"sensor_id"`
	MacAddress    string     `json:"mac_address"`
	Tipo          string     `json:"tipo"`
	LoteID        int        `json:"lote_id"`
	NombreLote    string     `json:"nombre_lote"`
	Conectado     bool       `json:"conectado"`
	UltimaLectura *time.Time `json:"ultima_lectura,omitempty"`
}

// EstadoSensoresResponse es la respuesta de GET /sensores/estado.
type EstadoSensoresResponse struct {
	Sensores []EstadoSensor `json:"sensores"`
}
