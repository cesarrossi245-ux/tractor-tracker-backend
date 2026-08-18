-- ============================================================
-- Sistema de tracking de tractores (IoT/GPS) — versión MySQL
-- Requiere MySQL 8.0+ (soporte de datos espaciales y UUID() como default)
-- ============================================================

-- ---------------------------------------------------------
-- Usuarios del sistema (operadores, administradores)
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id            CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    name          VARCHAR(120) NOT NULL,
    email         VARCHAR(180) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role          VARCHAR(20) NOT NULL DEFAULT 'operator', -- admin | operator
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ---------------------------------------------------------
-- Tractores y su dispositivo GPS asociado
-- device_key es el identificador único que manda el hardware
-- (por ejemplo el IMEI del GPS) para saber a qué tractor pertenece
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS tractors (
    id           CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    name         VARCHAR(120) NOT NULL,        -- "Tractor 01 - John Deere"
    plate        VARCHAR(20),
    brand        VARCHAR(60),
    model        VARCHAR(60),
    device_key   VARCHAR(80) UNIQUE NOT NULL,  -- IMEI o ID del GPS
    status       VARCHAR(20) NOT NULL DEFAULT 'active', -- active | inactive | maintenance
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ---------------------------------------------------------
-- Posiciones GPS (cada punto que manda el dispositivo)
-- Se usa tipo POINT (SRID 4326 = coordenadas GPS estándar)
-- para poder hacer cálculos de distancia y consultas espaciales.
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS positions (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    tractor_id    CHAR(36) NOT NULL,
    location      POINT NOT NULL SRID 4326,   -- guardado como POINT(lon, lat)
    speed_kmh     FLOAT,
    heading_deg   FLOAT,         -- dirección 0-360
    altitude_m    FLOAT,
    ignition_on   BOOLEAN DEFAULT false,
    fuel_level    FLOAT,         -- porcentaje 0-100, si el GPS lo reporta
    engine_hours  FLOAT,         -- horas motor acumuladas
    recorded_at   TIMESTAMP NOT NULL,          -- timestamp que reporta el dispositivo
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, -- cuando llegó al server

    CONSTRAINT fk_positions_tractor
        FOREIGN KEY (tractor_id) REFERENCES tractors(id) ON DELETE CASCADE,

    -- índice espacial: requiere que la columna sea NOT NULL (ya lo es)
    SPATIAL INDEX idx_positions_location (location)
);

-- Índice clave para la consulta más común: "recorrido de un tractor
-- en un rango de fechas" / "última posición conocida"
CREATE INDEX idx_positions_tractor_time
    ON positions (tractor_id, recorded_at DESC);

-- ---------------------------------------------------------
-- Geocercas (zonas del campo) - opcional, útil para alertas
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS geofences (
    id         CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    name       VARCHAR(120) NOT NULL,
    area       POLYGON NOT NULL SRID 4326,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    SPATIAL INDEX idx_geofences_area (area)
);
