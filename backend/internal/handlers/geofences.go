package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"tractor-tracker/backend/internal/models"
)

type GeofenceHandler struct {
	DB *sql.DB
}

func NewGeofenceHandler(db *sql.DB) *GeofenceHandler {
	return &GeofenceHandler{DB: db}
}

// geoJSONPolygon es la forma mínima del GeoJSON que devuelve MySQL con
// ST_AsGeoJSON() para una columna POLYGON. Las coordenadas vienen como
// [lon, lat] (GeoJSON siempre pone longitud primero).
type geoJSONPolygon struct {
	Type        string        `json:"type"`
	Coordinates [][][]float64 `json:"coordinates"`
}

// List devuelve todos los lotes guardados.
// GET /api/v1/geofences
func (h *GeofenceHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.DB.QueryContext(ctx, `
		SELECT id, name, ST_AsGeoJSON(area), created_at
		FROM geofences
		ORDER BY created_at DESC`)
	if err != nil {
		http.Error(w, "error consultando lotes", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var out []models.Geofence
	for rows.Next() {
		var g models.Geofence
		var rawGeoJSON string
		if err := rows.Scan(&g.ID, &g.Name, &rawGeoJSON, &g.CreatedAt); err != nil {
			http.Error(w, "error leyendo lotes", http.StatusInternalServerError)
			return
		}

		points, err := parseGeoJSONPolygon(rawGeoJSON)
		if err != nil {
			http.Error(w, "error interpretando geometría", http.StatusInternalServerError)
			return
		}
		g.Points = points
		out = append(out, g)
	}

	writeJSON(w, out)
}

// Create guarda un lote nuevo a partir del polígono dibujado en el mapa.
// POST /api/v1/geofences
func (h *GeofenceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateGeofenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "payload inválido", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		http.Error(w, "el lote necesita un nombre", http.StatusBadRequest)
		return
	}
	if len(req.Points) < 3 {
		http.Error(w, "un lote necesita al menos 3 puntos", http.StatusBadRequest)
		return
	}

	wkt := toWKTPolygon(req.Points)
	id := newUUID()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Nota: MySQL no soporta "INSERT ... RETURNING" (eso es sintaxis de
	// Postgres/MariaDB), así que generamos el id en Go y lo insertamos
	// explícito, en vez de depender del DEFAULT (UUID()) de la columna.
	_, err := h.DB.ExecContext(ctx, `
		INSERT INTO geofences (id, name, area)
		VALUES (?, ?, ST_GeomFromText(?, 4326))`, id, name, wkt,
	)
	if err != nil {
		http.Error(w, "error guardando el lote", http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"id": id})
}

// toWKTPolygon arma el texto "POLYGON((lon lat, lon lat, ...))" que
// espera MySQL, cerrando el anillo (el primer punto tiene que
// repetirse al final) si el frontend no lo hizo ya.
func toWKTPolygon(points []models.GeofencePoint) string {
	pts := points
	first, last := pts[0], pts[len(pts)-1]
	if first.Lat != last.Lat || first.Lon != last.Lon {
		pts = append(pts, first)
	}

	parts := make([]string, len(pts))
	for i, p := range pts {
		parts[i] = fmt.Sprintf("%f %f", p.Lon, p.Lat)
	}

	return "POLYGON((" + strings.Join(parts, ", ") + "))"
}

// newUUID genera un UUID v4 sin depender de ningún paquete externo
// (para no tener que tocar go.mod / go.sum, que requiere `go mod
// tidy` con conexión a internet en la máquina que compila).
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // versión 4
	b[8] = (b[8] & 0x3f) | 0x80 // variante RFC 4122
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func parseGeoJSONPolygon(raw string) ([]models.GeofencePoint, error) {
	var g geoJSONPolygon
	if err := json.Unmarshal([]byte(raw), &g); err != nil {
		return nil, err
	}
	if len(g.Coordinates) == 0 {
		return nil, fmt.Errorf("geometría vacía")
	}

	ring := g.Coordinates[0] // primer anillo = contorno exterior
	points := make([]models.GeofencePoint, len(ring))
	for i, coord := range ring {
		if len(coord) < 2 {
			continue
		}
		points[i] = models.GeofencePoint{Lon: coord[0], Lat: coord[1]}
	}

	return points, nil
}
