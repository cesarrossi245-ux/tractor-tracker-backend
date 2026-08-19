package models

import "time"

// User representa un operador con acceso al panel.
type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // nunca se serializa hacia el cliente
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

// LoginRequest es el payload que manda el frontend al iniciar sesión.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse es lo que devuelve el backend tras un login exitoso.
type LoginResponse struct {
	Token string `json:"token"`
	User  struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
		Role  string `json:"role"`
	} `json:"user"`
}

// Tractor representa una máquina registrada en el sistema.
type Tractor struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Plate     string    `json:"plate"`
	Brand     string    `json:"brand"`
	Model     string    `json:"model"`
	DeviceKey string    `json:"device_key"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// GPSData es el payload que manda el dispositivo GPS (o el simulador)
// cada vez que reporta una posición.
type GPSData struct {
	DeviceKey   string    `json:"device_key"`
	Latitude    float64   `json:"lat"`
	Longitude   float64   `json:"lon"`
	SpeedKmh    float64   `json:"speed_kmh"`
	HeadingDeg  float64   `json:"heading_deg"`
	AltitudeM   float64   `json:"altitude_m"`
	IgnitionOn  bool      `json:"ignition_on"`
	FuelLevel   float64   `json:"fuel_level"`
	EngineHours float64   `json:"engine_hours"`
	RecordedAt  time.Time `json:"recorded_at"`
}

// Position es un punto ya guardado en base de datos, listo
// para devolver al frontend.
type Position struct {
	ID          int64     `json:"id"`
	TractorID   string    `json:"tractor_id"`
	Latitude    float64   `json:"lat"`
	Longitude   float64   `json:"lon"`
	SpeedKmh    float64   `json:"speed_kmh"`
	HeadingDeg  float64   `json:"heading_deg"`
	AltitudeM   float64   `json:"altitude_m"`
	IgnitionOn  bool      `json:"ignition_on"`
	FuelLevel   float64   `json:"fuel_level"`
	EngineHours float64   `json:"engine_hours"`
	RecordedAt  time.Time `json:"recorded_at"`
}

// GeofencePoint es un vértice (lat/lon) del contorno de un lote.
type GeofencePoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Geofence representa un lote/parcela dibujado sobre el mapa.
type Geofence struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Points    []GeofencePoint `json:"points"`
	CreatedAt time.Time       `json:"created_at"`
}

// CreateGeofenceRequest es el payload que manda el frontend al guardar
// un lote nuevo: nombre y la lista de vértices en el orden en que se
// dibujaron.
type CreateGeofenceRequest struct {
	Name   string          `json:"name"`
	Points []GeofencePoint `json:"points"`
}

// LiveUpdate es lo que se transmite por WebSocket a los clientes
// conectados cuando llega una posición nueva.
type LiveUpdate struct {
	TractorID   string    `json:"tractor_id"`
	TractorName string    `json:"tractor_name"`
	Latitude    float64   `json:"lat"`
	Longitude   float64   `json:"lon"`
	SpeedKmh    float64   `json:"speed_kmh"`
	HeadingDeg  float64   `json:"heading_deg"`
	IgnitionOn  bool      `json:"ignition_on"`
	RecordedAt  time.Time `json:"recorded_at"`
}
