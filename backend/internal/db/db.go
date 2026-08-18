package db

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

// Connect abre un pool de conexiones a MySQL.
// dsn ejemplo: user:password@tcp(localhost:3306)/tractor_tracker?parseTime=true
//
// IMPORTANTE: parseTime=true es obligatorio, es lo que hace que el
// driver convierta las columnas TIMESTAMP/DATETIME en time.Time de Go
// automáticamente.
func Connect(dsn string) (*sql.DB, error) {
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir la conexión: %w", err)
	}

	conn.SetMaxOpenConns(20)
	conn.SetMaxIdleConns(10)

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("no se pudo conectar a la base de datos: %w", err)
	}

	return conn, nil
}
