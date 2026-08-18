package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"tractor-tracker/backend/internal/auth"
	"tractor-tracker/backend/internal/models"
)

type AuthHandler struct {
	DB *sql.DB
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{DB: db}
}

// Login valida email + contraseña contra la tabla `users` y, si son
// correctos, devuelve un JWT que el frontend debe mandar en cada
// petición posterior (header Authorization: Bearer <token>).
// POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		http.Error(w, "email y password son requeridos", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var user models.User
	err := h.DB.QueryRowContext(ctx,
		`SELECT id, name, email, password_hash, role FROM users WHERE email = ?`,
		req.Email,
	).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.Role)

	// Importante: devolvemos el mismo error genérico tanto si el
	// usuario no existe como si la contraseña no coincide, para no
	// revelar a un atacante qué correos están registrados.
	invalidCreds := func() {
		http.Error(w, "credenciales inválidas", http.StatusUnauthorized)
	}

	if err != nil {
		invalidCreds()
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		invalidCreds()
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		http.Error(w, "error generando la sesión", http.StatusInternalServerError)
		return
	}

	var resp models.LoginResponse
	resp.Token = token
	resp.User.ID = user.ID
	resp.User.Name = user.Name
	resp.User.Email = user.Email
	resp.User.Role = user.Role

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
