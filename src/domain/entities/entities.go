//src/domain/entities/entities.go
// Package entities contiene los modelos de dominio del WebSocket Gateway.
package entities

import "time"

// LecturaDTO es la forma de una lectura (histórica o en vivo) que viaja
// hacia el cliente WebSocket. Debe coincidir EXACTAMENTE con
// ingesta-iot/domain/entities.LecturaAmbientalDTO, porque las lecturas en
// vivo llegan tal cual desde Redis en ese formato — este Gateway solo las
// reenvía, no las reconstruye.
//
// Alineado con la migración de BD: humedad y la vieja lluvia (float 0-1)
// ya no existen; ahora son lluvia_analog (lectura cruda del ADC) y
// lluvia_detectada (boolean). humedad_grano pasó a int16 porque también es
// una lectura cruda del ADC (0-4095), no un porcentaje.
type LecturaDTO struct {
	Temperatura      *float64 `json:"temperatura,omitempty"`
	TemperaturaGrano *float64 `json:"temperatura_grano,omitempty"`
	Luz              *float64 `json:"luz,omitempty"`
	LluviaAnalog     *int16   `json:"lluvia_analog,omitempty"`
	LluviaDetectada  *bool    `json:"lluvia_detectada,omitempty"`
	HumedadGrano     *int16   `json:"humedad_grano,omitempty"`
	PresionHpa       *float64 `json:"presion_hpa,omitempty"`
	AltitudM         *float64 `json:"altitud_m,omitempty"`
	// EstadoSensores solo llega en lecturas en vivo (ingesta-iot lo manda
	// tal cual del ESP32, ver ingesta-iot/domain/entities.EstadoSensores).
	// En el histórico (GetUltimas, lectura_repository.go) siempre queda en
	// nil/omitido porque no existe columna para esto en Postgres — no se
	// persiste a propósito.
	EstadoSensores *EstadoSensores `json:"estado_sensores,omitempty"`
}

// EstadoSensores refleja qué sensores físicos del ESP32 están conectados,
// tal como lo reporta el propio dispositivo junto con la lectura. Debe
// coincidir con ingesta-iot/domain/entities.EstadoSensores.
type EstadoSensores struct {
	Bmp280       *bool `json:"bmp280,omitempty"`
	Ds18b20      *bool `json:"ds18b20,omitempty"`
	Bh1750       *bool `json:"bh1750,omitempty"`
	Fc37         *bool `json:"fc37,omitempty"`
	HumedadSuelo *bool `json:"humedad_suelo,omitempty"`
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

// ResumenLote es la respuesta de GET /lotes/{id}/resumen: agregados
// (min/promedio/max) calculados sobre TODAS las lecturas guardadas del
// lote, no solo las últimas N como el histórico del WebSocket
// (GetUltimas). Pensado para la vista de Monitoreo — a diferencia de
// Tiempo Real (últimos ~60 puntos) y Sensores (conectado/desconectado),
// esta es la única vista que responde "¿cómo va el lote completo?". No
// depende de api-mobile: se calcula directo sobre Postgres desde acá.
type ResumenLote struct {
	TotalLecturas       int        `json:"total_lecturas"`
	PrimeraLectura      *time.Time `json:"primera_lectura,omitempty"`
	UltimaLectura       *time.Time `json:"ultima_lectura,omitempty"`
	TemperaturaMin       *float64  `json:"temperatura_min,omitempty"`
	TemperaturaProm      *float64  `json:"temperatura_prom,omitempty"`
	TemperaturaMax       *float64  `json:"temperatura_max,omitempty"`
	TemperaturaGranoMin  *float64  `json:"temperatura_grano_min,omitempty"`
	TemperaturaGranoProm *float64  `json:"temperatura_grano_prom,omitempty"`
	TemperaturaGranoMax  *float64  `json:"temperatura_grano_max,omitempty"`
	HumedadGranoMin      *float64  `json:"humedad_grano_min,omitempty"`
	HumedadGranoProm     *float64  `json:"humedad_grano_prom,omitempty"`
	HumedadGranoMax      *float64  `json:"humedad_grano_max,omitempty"`
	PresionHpaMin        *float64  `json:"presion_hpa_min,omitempty"`
	PresionHpaProm       *float64  `json:"presion_hpa_prom,omitempty"`
	PresionHpaMax        *float64  `json:"presion_hpa_max,omitempty"`
}