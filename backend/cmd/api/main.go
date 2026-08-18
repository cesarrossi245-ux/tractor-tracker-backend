package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"tractor-tracker/backend/internal/auth"
	"tractor-tracker/backend/internal/db"
	"tractor-tracker/backend/internal/handlers"
	"tractor-tracker/backend/internal/ws"
)

func main() {
	dsn := getEnv("DATABASE_URL", "root:root@tcp(localhost:3306)/tractor_tracker?parseTime=true")
	port := getEnv("PORT", "8080")

	pool, err := db.Connect(dsn)
	if err != nil {
		log.Fatalf("no se pudo conectar a la base de datos: %v", err)
	}
	defer pool.Close()
	log.Println("conectado a MySQL")

	hub := ws.NewHub()
	gpsHandler := handlers.NewGPSHandler(pool, hub)
	tractorHandler := handlers.NewTractorHandler(pool)
	authHandler := handlers.NewAuthHandler(pool)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	r.Route("/api/v1", func(r chi.Router) {
		// Rutas públicas: el dispositivo GPS no inicia sesión (se
		// identifica con su device_key), y el login obviamente no
		// puede requerir estar ya autenticado.
		r.Post("/gps/ingest", gpsHandler.Ingest)
		r.Post("/auth/login", authHandler.Login)

		// Rutas protegidas: requieren un JWT válido (el operador
		// tiene que haber iniciado sesión desde el frontend).
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth)
			r.Get("/tractors", tractorHandler.List)
			r.Get("/tractors/{id}/last", tractorHandler.LastPosition)
			r.Get("/tractors/{id}/positions", tractorHandler.History)
		})
	})

	// WebSocket para posiciones en tiempo real (protegido: el token
	// viaja como ?token=... en la URL, ver internal/auth/middleware.go)
	r.With(auth.RequireAuth).Get("/ws", hub.ServeHTTP)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
		r.Get("/tracker", handlers.TrackerPage)

	// 📍 PÁGINA DEL TRACKER GPS (sirve HTML con geolocalización)
	r.Get("/tracker", handlers.TrackerPage)

	log.Printf("servidor escuchando en :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}