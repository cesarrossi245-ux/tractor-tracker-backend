package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"tractor-tracker/backend/internal/models"
	"tractor-tracker/backend/internal/ws"
)

type GPSHandler struct {
	DB  *sql.DB
	Hub *ws.Hub
}

func NewGPSHandler(db *sql.DB, hub *ws.Hub) *GPSHandler {
	return &GPSHandler{DB: db, Hub: hub}
}

// Ingest recibe el payload de un dispositivo GPS (o del simulador)
// POST /api/v1/gps/ingest
func (h *GPSHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	var data models.GPSData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	if data.DeviceKey == "" {
		http.Error(w, "device_key es requerido", http.StatusBadRequest)
		return
	}
	if data.RecordedAt.IsZero() {
		data.RecordedAt = time.Now().UTC()
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Buscamos a qué tractor pertenece este dispositivo
	var tractorID, tractorName string
	err := h.DB.QueryRowContext(ctx,
		`SELECT id, name FROM tractors WHERE device_key = ?`,
		data.DeviceKey,
	).Scan(&tractorID, &tractorName)
	if err != nil {
		http.Error(w, "dispositivo no registrado (device_key desconocido)", http.StatusNotFound)
		return
	}

	// Insertamos la posición. ST_SRID(POINT(lon, lat), 4326) -- ¡ojo con el orden!
	_, err = h.DB.ExecContext(ctx, `
		INSERT INTO positions
			(tractor_id, location, speed_kmh, heading_deg, altitude_m,
			 ignition_on, fuel_level, engine_hours, recorded_at)
		VALUES
			(?, ST_SRID(POINT(?, ?), 4326),
			 ?, ?, ?, ?, ?, ?, ?)`,
		tractorID, data.Longitude, data.Latitude,
		data.SpeedKmh, data.HeadingDeg, data.AltitudeM,
		data.IgnitionOn, data.FuelLevel, data.EngineHours, data.RecordedAt,
	)
	if err != nil {
		log.Printf("gps: error insertando posición: %v", err)
		http.Error(w, "error guardando la posición", http.StatusInternalServerError)
		return
	}

	// Transmitimos en vivo a todos los navegadores conectados por WebSocket
	h.Hub.Broadcast(models.LiveUpdate{
		TractorID:   tractorID,
		TractorName: tractorName,
		Latitude:    data.Latitude,
		Longitude:   data.Longitude,
		SpeedKmh:    data.SpeedKmh,
		HeadingDeg:  data.HeadingDeg,
		IgnitionOn:  data.IgnitionOn,
		RecordedAt:  data.RecordedAt,
	})

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
