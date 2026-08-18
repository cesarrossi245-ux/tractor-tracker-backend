// Comando para crear un usuario con contraseña encriptada.
// Es necesario porque todavía no hay un panel de administración
// de usuarios — este es el "primer usuario" para poder iniciar sesión.
//
// Uso:
//
//	go run ./cmd/createuser -name "César" -email cesar@ejemplo.com -password "algo-seguro"
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"

	"tractor-tracker/backend/internal/db"
)

func main() {
	name := flag.String("name", "", "Nombre del usuario")
	email := flag.String("email", "", "Correo (se usa para iniciar sesión)")
	password := flag.String("password", "", "Contraseña en texto plano (se encripta antes de guardar)")
	role := flag.String("role", "admin", "Rol: admin u operator")
	flag.Parse()

	if *name == "" || *email == "" || *password == "" {
		fmt.Println("Uso: go run ./cmd/createuser -name \"Tu Nombre\" -email tu@correo.com -password \"tu-contraseña\"")
		os.Exit(1)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("falta la variable de entorno DATABASE_URL")
	}

	pool, err := db.Connect(dsn)
	if err != nil {
		log.Fatalf("no se pudo conectar a la base de datos: %v", err)
	}
	defer pool.Close()

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("error encriptando la contraseña: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = pool.ExecContext(ctx, `
		INSERT INTO users (name, email, password_hash, role)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE password_hash = VALUES(password_hash), name = VALUES(name)`,
		*name, *email, string(hash), *role,
	)
	if err != nil {
		log.Fatalf("error guardando el usuario: %v", err)
	}

	fmt.Printf("Usuario creado/actualizado: %s (%s), rol: %s\n", *name, *email, *role)
}
