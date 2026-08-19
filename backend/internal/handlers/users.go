package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"tractor-tracker/backend/internal/auth"
	"tractor-tracker/backend/internal/models"
)

type UserHandler struct {
	DB *sql.DB
}

func NewUserHandler(db *sql.DB) *UserHandler {
	return &UserHandler{DB: db}
}

// List devuelve todos los usuarios del sistema (sin el hash de
// contraseña — models.User ya lo marca como `json:"-"`).
// GET /api/v1/users
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.DB.QueryContext(ctx, `
		SELECT id, name, email, role, created_at
		FROM users
		ORDER BY name`)
	if err != nil {
		http.Error(w, "error consultando usuarios", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt); err != nil {
			http.Error(w, "error leyendo usuarios", http.StatusInternalServerError)
			return
		}
		users = append(users, u)
	}

	writeJSON(w, users)
}

// createUserRequest es el payload para dar de alta un operador nuevo.
type createUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// Create da de alta un usuario nuevo. Solo lo puede hacer un admin
// (se valida con RequireRole en las rutas, ver main.go).
// POST /api/v1/users
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "payload inválido", http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Name == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "nombre, email y contraseña son requeridos", http.StatusBadRequest)
		return
	}
	if req.Role != "admin" && req.Role != "operator" {
		req.Role = "operator"
	}
	if len(req.Password) < 6 {
		http.Error(w, "la contraseña debe tener al menos 6 caracteres", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "error preparando la contraseña", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_, err = h.DB.ExecContext(ctx, `
		INSERT INTO users (id, name, email, password_hash, role)
		VALUES (UUID(), ?, ?, ?, ?)`,
		req.Name, req.Email, string(hash), req.Role,
	)
	if err != nil {
		// El email es UNIQUE en el schema; un fallo acá casi siempre
		// significa que ya existe un usuario con ese correo.
		http.Error(w, "no se pudo crear el usuario (¿el email ya existe?)", http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]string{"status": "creado"})
}

// updateRoleRequest es el payload para cambiar el rol de un usuario.
type updateRoleRequest struct {
	Role string `json:"role"`
}

// UpdateRole cambia el rol (admin | operator) de un usuario existente.
// PUT /api/v1/users/{id}/role
func (h *UserHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "payload inválido", http.StatusBadRequest)
		return
	}
	if req.Role != "admin" && req.Role != "operator" {
		http.Error(w, "rol inválido: debe ser 'admin' u 'operator'", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res, err := h.DB.ExecContext(ctx, `UPDATE users SET role = ? WHERE id = ?`, req.Role, id)
	if err != nil {
		http.Error(w, "error actualizando el rol", http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		http.Error(w, "usuario no encontrado", http.StatusNotFound)
		return
	}

	writeJSON(w, map[string]string{"status": "actualizado"})
}

// RequireAdmin es un middleware adicional (se usa junto a
// auth.RequireAuth) que solo deja pasar a usuarios con rol "admin".
// Pensado para las rutas de gestión de usuarios, que un operador
// normal no debería poder tocar.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := auth.ClaimsFromContext(r.Context())
		if claims == nil || claims.Role != "admin" {
			http.Error(w, "se requiere rol de administrador", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
