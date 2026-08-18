// Simulador de tractores con GPS.
// Manda posiciones falsas al API cada pocos segundos, simulando
// el comportamiento real de un dispositivo GPS instalado en campo.
//
// Uso:
//
//	go run ./cmd/simulator
//
// Variables de entorno opcionales:
//
//	API_URL   (default: http://localhost:8080/api/v1/gps/ingest)
//	INTERVAL  segundos entre envíos (default: 5)
package main

import (
	"bytes"
	"encoding/json"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

type gpsPayload struct {
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

// simTractor representa un tractor virtual que se mueve
// en pequeños círculos/zigzag alrededor de un punto de origen,
// simulando el trabajo en un campo.
type simTractor struct {
	DeviceKey   string
	Lat, Lon    float64
	Heading     float64
	FuelLevel   float64
	EngineHours float64
}

func main() {
	apiURL := getEnv("API_URL", "http://localhost:8080/api/v1/gps/ingest")
	intervalSec, _ := strconv.Atoi(getEnv("INTERVAL", "5"))
	interval := time.Duration(intervalSec) * time.Second

	// IMPORTANTE: estos device_key deben coincidir con los que
	// registres en la tabla `tractors` (columna device_key).
	tractors := []*simTractor{
		{DeviceKey: "GPS-0001", Lat: -14.0678, Lon: -75.7286, FuelLevel: 90, EngineHours: 120}, // Ica, Perú
		{DeviceKey: "GPS-0002", Lat: -14.0700, Lon: -75.7320, FuelLevel: 75, EngineHours: 340},
		{DeviceKey: "GPS-0003", Lat: -14.0650, Lon: -75.7250, FuelLevel: 55, EngineHours: 890},
	}

	log.Printf("simulador iniciado: %d tractores, enviando a %s cada %s", len(tractors), apiURL, interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		for _, t := range tractors {
			moveTractor(t)
			sendPosition(apiURL, t)
		}
	}
}

// moveTractor avanza el tractor un pequeño paso en su dirección
// actual, con variaciones aleatorias de rumbo (simula trabajo en surcos).
func moveTractor(t *simTractor) {
	t.Heading += (rand.Float64() - 0.5) * 20 // pequeño giro aleatorio
	if t.Heading < 0 {
		t.Heading += 360
	}
	if t.Heading >= 360 {
		t.Heading -= 360
	}

	// ~0.00005 grados por tick ≈ unos metros; ajusta según INTERVAL
	step := 0.00005
	rad := t.Heading * math.Pi / 180
	t.Lat += step * math.Cos(rad)
	t.Lon += step * math.Sin(rad)

	t.FuelLevel -= 0.05
	if t.FuelLevel < 0 {
		t.FuelLevel = 0
	}
	t.EngineHours += 0.001
}

func sendPosition(apiURL string, t *simTractor) {
	payload := gpsPayload{
		DeviceKey:   t.DeviceKey,
		Latitude:    t.Lat,
		Longitude:   t.Lon,
		SpeedKmh:    8 + rand.Float64()*6, // velocidad típica de trabajo en campo
		HeadingDeg:  t.Heading,
		AltitudeM:   400 + rand.Float64()*10,
		IgnitionOn:  true,
		FuelLevel:   t.FuelLevel,
		EngineHours: t.EngineHours,
		RecordedAt:  time.Now().UTC(),
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(apiURL, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("[%s] error enviando posición: %v", t.DeviceKey, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Printf("[%s] servidor respondió %d", t.DeviceKey, resp.StatusCode)
		return
	}

	log.Printf("[%s] posición enviada: lat=%.6f lon=%.6f", t.DeviceKey, t.Lat, t.Lon)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
