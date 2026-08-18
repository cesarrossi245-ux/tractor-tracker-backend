package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"tractor-tracker/backend/internal/models"
)

type TractorHandler struct {
	DB *sql.DB
}

func NewTractorHandler(db *sql.DB) *TractorHandler {
	return &TractorHandler{DB: db}
}

// List devuelve todos los tractores registrados.
// GET /api/v1/tractors
func (h *TractorHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.DB.QueryContext(ctx, `
		SELECT id, name, plate, brand, model, device_key, status, created_at
		FROM tractors
		ORDER BY name`)
	if err != nil {
		http.Error(w, "error consultando tractores", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tractors []models.Tractor
	for rows.Next() {
		var t models.Tractor
		if err := rows.Scan(&t.ID, &t.Name, &t.Plate, &t.Brand, &t.Model, &t.DeviceKey, &t.Status, &t.CreatedAt); err != nil {
			http.Error(w, "error leyendo tractores", http.StatusInternalServerError)
			return
		}
		tractors = append(tractors, t)
	}

	writeJSON(w, tractors)
}

// LastPosition devuelve la última posición conocida de un tractor.
// GET /api/v1/tractors/{id}/last
func (h *TractorHandler) LastPosition(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var p models.Position
	err := h.DB.QueryRowContext(ctx, `
		SELECT id, tractor_id,
		       ST_Y(location) AS lat,
		       ST_X(location) AS lon,
		       speed_kmh, heading_deg, altitude_m,
		       ignition_on, fuel_level, engine_hours, recorded_at
		FROM positions
		WHERE tractor_id = ?
		ORDER BY recorded_at DESC
		LIMIT 1`, id,
	).Scan(&p.ID, &p.TractorID, &p.Latitude, &p.Longitude, &p.SpeedKmh,
		&p.HeadingDeg, &p.AltitudeM, &p.IgnitionOn, &p.FuelLevel, &p.EngineHours, &p.RecordedAt)
	if err != nil {
		http.Error(w, "no hay posiciones para este tractor", http.StatusNotFound)
		return
	}

	writeJSON(w, p)
}

// History devuelve el recorrido de un tractor entre dos fechas.
// GET /api/v1/tractors/{id}/positions?from=RFC3339&to=RFC3339
func (h *TractorHandler) History(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	from, to, err := parseRange(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	rows, err := h.DB.QueryContext(ctx, `
		SELECT id, tractor_id,
		       ST_Y(location) AS lat,
		       ST_X(location) AS lon,
		       speed_kmh, heading_deg, altitude_m,
		       ignition_on, fuel_level, engine_hours, recorded_at
		FROM positions
		WHERE tractor_id = ? AND recorded_at BETWEEN ? AND ?
		ORDER BY recorded_at ASC`, id, from, to,
	)
	if err != nil {
		http.Error(w, "error consultando el recorrido", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var positions []models.Position
	for rows.Next() {
		var p models.Position
		if err := rows.Scan(&p.ID, &p.TractorID, &p.Latitude, &p.Longitude, &p.SpeedKmh,
			&p.HeadingDeg, &p.AltitudeM, &p.IgnitionOn, &p.FuelLevel, &p.EngineHours, &p.RecordedAt); err != nil {
			http.Error(w, "error leyendo el recorrido", http.StatusInternalServerError)
			return
		}
		positions = append(positions, p)
	}

	writeJSON(w, positions)
}

func parseRange(r *http.Request) (time.Time, time.Time, error) {
	q := r.URL.Query()
	toStr := q.Get("to")
	fromStr := q.Get("from")

	to := time.Now().UTC()
	if toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		to = t
	}

	from := to.Add(-24 * time.Hour)
	if fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		from = t
	}

	return from, to, nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
