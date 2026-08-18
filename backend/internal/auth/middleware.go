package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const claimsContextKey contextKey = "claims"

// RequireAuth es un middleware de chi que exige un header
// "Authorization: Bearer <token>" válido. Si falta o es inválido,
// corta la petición con 401. Si es válido, guarda los claims en el
// contexto para que los handlers los puedan leer con ClaimsFromContext.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")

		// El WebSocket del navegador no puede mandar headers custom,
		// así que también aceptamos el token como query param: ?token=...
		tokenString := strings.TrimPrefix(header, "Bearer ")
		if tokenString == header { // no tenía el prefijo "Bearer "
			tokenString = r.URL.Query().Get("token")
		}

		if tokenString == "" {
			http.Error(w, "no autorizado: falta token", http.StatusUnauthorized)
			return
		}

		claims, err := ParseToken(tokenString)
		if err != nil {
			http.Error(w, "no autorizado: token inválido o expirado", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), claimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ClaimsFromContext recupera los claims del usuario autenticado
// dentro de un handler protegido por RequireAuth.
func ClaimsFromContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsContextKey).(*Claims)
	return claims
}
