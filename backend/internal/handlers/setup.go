package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// SetupHandler expone una ruta temporal, pensada solo para crear el
// primer usuario admin sin necesitar Go/Node instalados localmente ni
// acceso directo a la base de datos.
//
// IMPORTANTE: es un mecanismo de arranque, no debe quedar activo en
// producción. Una vez creado el primer usuario, borra este archivo,
// quita la ruta de cmd/api/main.go y vuelve a desplegar.
type SetupHandler struct {
	DB *sql.DB
}

func NewSetupHandler(db *sql.DB) *SetupHandler {
	return &SetupHandler{DB: db}
}

// CreateFirstUser crea o actualiza un usuario a partir de query params.
// GET /api/v1/setup/create-user?secret=...&name=...&email=...&password=...&role=admin
func (h *SetupHandler) CreateFirstUser(w http.ResponseWriter, r *http.Request) {
	expected := os.Getenv("SETUP_SECRET")
	if expected == "" {
		http.Error(w, "SETUP_SECRET no está configurado en el servidor", http.StatusForbidden)
		return
	}
	if r.URL.Query().Get("secret") != expected {
		http.Error(w, "clave secreta inválida", http.StatusForbidden)
		return
	}

	name := strings.TrimSpace(r.URL.Query().Get("name"))
	email := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("email")))
	password := r.URL.Query().Get("password")
	role := r.URL.Query().Get("role")
	if role != "admin" && role != "operator" {
		role = "admin"
	}

	if name == "" || email == "" || password == "" {
		http.Error(w, "faltan parámetros: name, email, password", http.StatusBadRequest)
		return
	}
	if len(password) < 6 {
		http.Error(w, "la contraseña debe tener al menos 6 caracteres", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "error preparando la contraseña", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_, err = h.DB.ExecContext(ctx, `
		INSERT INTO users (id, name, email, password_hash, role)
		VALUES (UUID(), ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE password_hash = VALUES(password_hash), name = VALUES(name), role = VALUES(role)`,
		name, email, string(hash), role,
	)
	if err != nil {
		http.Error(w, "error guardando el usuario: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Usuario creado/actualizado: " + name + " (" + email + "), rol: " + role +
		"\n\nAhora BORRA esta ruta de setup y vuelve a desplegar por seguridad."))
}
