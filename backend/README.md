# Tractor Tracker — Backend (Go + MySQL)

Backend del sistema de tracking IoT/GPS para tractores. Recibe datos de
ubicación de dispositivos GPS, los guarda en MySQL (con tipos espaciales
nativos), y los expone vía API REST + WebSocket en tiempo real.

## Requisitos

- Go 1.22+
- **MySQL 8.0+** (necesario para tipos `POINT`, funciones `ST_X`/`ST_Y`,
  índices `SPATIAL`, y `DEFAULT (UUID())`. Versiones anteriores como
  5.7 no soportan todo esto).

## 1. Preparar la base de datos

Puedes usar **HeidiSQL**, la terminal `mysql`, o cualquier cliente que
prefieras — el SQL es el mismo.

### Opción A: con HeidiSQL
1. Conéctate a tu servidor MySQL.
2. Crea una base de datos nueva llamada `tractor_tracker`.
3. Abre una pestaña de **Query**, selecciona esa base de datos, pega el
   contenido de `internal/db/schema.sql` y ejecútalo (F9).
4. Repite lo mismo con `internal/db/seed.sql` (datos de ejemplo, opcional).

### Opción B: por terminal
```bash
mysql -u root -p -e "CREATE DATABASE tractor_tracker"
mysql -u root -p tractor_tracker < internal/db/schema.sql
mysql -u root -p tractor_tracker < internal/db/seed.sql   # opcional
```

## 2. Configurar variables de entorno

```bash
cp .env.example .env
```

Edita `.env` con tus datos reales. El formato de conexión de MySQL es:
```
usuario:password@tcp(host:puerto)/nombre_basedatos?parseTime=true
```
`parseTime=true` es obligatorio — es lo que hace que las fechas se
lean automáticamente como `time.Time` en Go.

### Si usas MySQL en la nube (Aiven u otro proveedor administrado)

Copia el host, puerto, usuario y password desde el panel de conexión
de tu proveedor, y agrega `&tls=true` al final del DSN (la mayoría de
proveedores en la nube exigen conexión cifrada):

```
DATABASE_URL=avnadmin:TU_PASSWORD@tcp(tu-servicio.aivencloud.com:12345)/defaultdb?parseTime=true&tls=true
```

El nombre de la base por defecto en Aiven es `defaultdb` (no
`tractor_tracker` como en una instalación local) — puedes crear el
schema ahí mismo, o crear una base nueva llamada `tractor_tracker`
desde el editor SQL del panel de Aiven o desde HeidiSQL conectado
remotamente.

## 3. Instalar dependencias y compilar

Este proyecto usa módulos de Go que necesitan descargarse de
proxy.golang.org (no lo pude ejecutar yo en este entorno por
restricciones de red, así que hazlo tú la primera vez):

```bash
go mod tidy
```

## 4. Levantar el API

```bash
export $(cat .env | xargs)
go run ./cmd/api
```

Deberías ver:
```
conectado a MySQL
servidor escuchando en :8080
```

## 5. Levantar el simulador de GPS (en otra terminal)

Simula 3 tractores moviéndose y mandando su posición cada 5 segundos:

```bash
go run ./cmd/simulator
```

Si usaste `seed.sql`, los tractores del simulador (`GPS-0001`, `GPS-0002`,
`GPS-0003`) ya están registrados y las posiciones se guardarán solas.
Puedes verlas llegar en tiempo real abriendo la tabla `positions` en
HeidiSQL y refrescando.

## Endpoints disponibles

| Método | Ruta                              | Auth | Descripción                              |
|--------|------------------------------------|------|--------------------------------------------|
| POST   | `/api/v1/auth/login`               | No   | Login: `{email, password}` → JWT          |
| POST   | `/api/v1/gps/ingest`               | No   | Recibe una posición desde un dispositivo  |
| GET    | `/api/v1/tractors`                 | Sí   | Lista todos los tractores                 |
| GET    | `/api/v1/tractors/{id}/last`       | Sí   | Última posición conocida de un tractor    |
| GET    | `/api/v1/tractors/{id}/positions`  | Sí   | Recorrido histórico (`?from=&to=` RFC3339)|
| GET    | `/ws`                              | Sí   | WebSocket con posiciones en tiempo real   |
| GET    | `/health`                          | No   | Health check                              |

Las rutas marcadas "Sí" requieren mandar el JWT en el header
`Authorization: Bearer <token>` (el WebSocket, al no poder mandar
headers custom desde el navegador, recibe el token como
`?token=<token>` en la URL).

## Crear tu primer usuario

Como todavía no hay un panel para administrar usuarios, se crea por
línea de comandos:

```bash
go run ./cmd/createuser -name "Tu Nombre" -email tu@correo.com -password "una-contraseña-segura"
```

Esto guarda la contraseña ya encriptada (bcrypt) en la tabla `users`.
Puedes correrlo de nuevo con el mismo email para cambiar la
contraseña de ese usuario.

## Iniciar sesión

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "tu@correo.com", "password": "una-contraseña-segura"}'
```

Devuelve un JWT válido por 24 horas. Guárdalo y mándalo en cada
petición protegida:

```bash
curl http://localhost:8080/api/v1/tractors \
  -H "Authorization: Bearer TU_TOKEN_AQUI"
```

## Variable de entorno JWT_SECRET

Agrega `JWT_SECRET` a tu `.env` con un valor largo y aleatorio propio
(si no la defines, el backend usa un valor de desarrollo — está bien
para pruebas, pero cámbialo antes de usar esto en producción):

```
JWT_SECRET=una-cadena-larga-y-aleatoria-solo-tuya
```

### Ejemplo: enviar una posición manualmente

```bash
curl -X POST http://localhost:8080/api/v1/gps/ingest \
  -H "Content-Type: application/json" \
  -d '{
    "device_key": "GPS-0001",
    "lat": -14.0678,
    "lon": -75.7286,
    "speed_kmh": 12.5,
    "heading_deg": 90,
    "ignition_on": true,
    "fuel_level": 80,
    "engine_hours": 120.5
  }'
```

### Formato del mensaje en tiempo real (WebSocket)

Cada vez que llega una posición nueva, todos los clientes conectados a
`/ws` reciben un JSON como:

```json
{
  "tractor_id": "uuid...",
  "tractor_name": "Tractor 01",
  "lat": -14.0678,
  "lon": -75.7286,
  "speed_kmh": 12.5,
  "heading_deg": 90,
  "ignition_on": true,
  "recorded_at": "2026-08-14T10:00:00Z"
}
```

## Nota sobre MySQL más antiguo (5.7 o inferior)

Si estás forzado a usar una versión vieja de MySQL sin soporte espacial
completo, la alternativa es guardar `latitude` y `longitude` como dos
columnas `DECIMAL(10,7)` normales en vez de un tipo `POINT`. Se pierden
las consultas espaciales avanzadas (geocercas, "tractores a X metros de
un punto"), pero para mostrar el mapa y el recorrido histórico funciona
igual. Si es tu caso, avísame y adapto el esquema.

## Cuando conectes el hardware GPS real

En lugar del simulador, el dispositivo real deberá hacer un POST HTTP
igual al de `cmd/simulator` hacia `/api/v1/gps/ingest`. Si tu GPS usa
un protocolo distinto (MQTT, TCP binario propietario tipo Teltonika o
Concox), se necesita un pequeño servicio "traductor" que reciba ese
protocolo y llame a este mismo endpoint — lo armamos cuando definas
el modelo exacto de hardware.

## Siguiente paso

Frontend en React con mapa en tiempo real (Leaflet) que consuma
`GET /api/v1/tractors`, `GET /api/v1/tractors/{id}/positions` y se
conecte al WebSocket `/ws`.
