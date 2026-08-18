package handlers

import "net/http"

// TrackerPage sirve una página web simple y autocontenida (sin
// dependencias externas) pensada para abrirse desde el navegador de
// un celular. Usa la API de geolocalización del navegador para
// mandar la posición real del teléfono al mismo endpoint de ingesta
// que usa el simulador (/api/v1/gps/ingest) — así puedes probar el
// sistema completo con movimiento GPS real, sin comprar hardware
// todavía.
// GET /tracker
func TrackerPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(trackerHTML))
}

const trackerHTML = `<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0">
<title>Tracker GPS — Tractor Tracker</title>
<style>
  :root {
    --bg: #12140f; --panel: #1b1f16; --border: #2a2f22;
    --text: #edeae1; --text-dim: #a8ac9c; --amber: #e8a33d;
    --green: #8fb339; --rust: #c1503f;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0; min-height: 100vh; background: var(--bg); color: var(--text);
    font-family: -apple-system, system-ui, sans-serif;
    display: flex; align-items: center; justify-content: center; padding: 20px;
  }
  .card {
    width: 100%; max-width: 380px; background: var(--panel);
    border: 1px solid var(--border); border-radius: 8px; padding: 24px;
  }
  h1 { font-size: 17px; margin: 0 0 4px; }
  .sub { font-size: 12px; color: var(--text-dim); margin: 0 0 20px; text-transform: uppercase; letter-spacing: .05em; }
  label { display: block; font-size: 11px; color: var(--text-dim); text-transform: uppercase; letter-spacing: .05em; margin-bottom: 6px; }
  input {
    width: 100%; background: var(--bg); border: 1px solid var(--border); border-radius: 4px;
    color: var(--text); font-size: 15px; padding: 10px 12px; margin-bottom: 16px;
  }
  button {
    width: 100%; border: none; border-radius: 4px; padding: 14px; font-size: 15px;
    font-weight: 600; cursor: pointer;
  }
  #startBtn { background: var(--amber); color: #17140c; }
  #stopBtn { background: var(--rust); color: #fff; display: none; }
  .status {
    margin-top: 20px; padding-top: 16px; border-top: 1px solid var(--border);
    font-family: 'SFMono-Regular', Menlo, monospace; font-size: 13px; line-height: 1.9;
  }
  .status span { color: var(--text-dim); }
  .status b { float: right; }
  .dot {
    display: inline-block; width: 8px; height: 8px; border-radius: 50%;
    background: var(--text-dim); margin-right: 6px;
  }
  .dot.on { background: var(--green); box-shadow: 0 0 6px var(--green); }
  .dot.err { background: var(--rust); }
  #msg { font-size: 12px; margin-top: 12px; color: var(--text-dim); }
</style>
</head>
<body>
<div class="card">
  <h1>Tracker GPS</h1>
  <p class="sub">Manda tu ubicación real al panel de flota</p>

  <label for="deviceKey">Device key (creado en la base de datos)</label>
  <input id="deviceKey" placeholder="PHONE-001" value="PHONE-001">

  <button id="startBtn">Iniciar tracking</button>
  <button id="stopBtn">Detener tracking</button>

  <div class="status">
    <div><span class="dot" id="statusDot"></span><span>Estado</span><b id="statusText">Detenido</b></div>
    <div><span>Latitud</span><b id="lat">—</b></div>
    <div><span>Longitud</span><b id="lon">—</b></div>
    <div><span>Velocidad</span><b id="speed">—</b></div>
    <div><span>Precisión</span><b id="acc">—</b></div>
    <div><span>Envíos</span><b id="count">0</b></div>
  </div>
  <p id="msg"></p>
</div>

<script>
let watchId = null;
let sendCount = 0;
let lastSend = 0;
const MIN_INTERVAL_MS = 4000; // no mandar más seguido que cada 4s

const el = (id) => document.getElementById(id);

function setStatus(text, cls) {
  el('statusText').textContent = text;
  el('statusDot').className = 'dot' + (cls ? ' ' + cls : '');
}

async function sendPosition(pos) {
  const now = Date.now();
  if (now - lastSend < MIN_INTERVAL_MS) return;
  lastSend = now;

  const c = pos.coords;
  const deviceKey = el('deviceKey').value.trim();
  if (!deviceKey) return;

  const speedKmh = c.speed != null ? c.speed * 3.6 : 0;
  const headingDeg = c.heading != null && !isNaN(c.heading) ? c.heading : 0;

  el('lat').textContent = c.latitude.toFixed(6);
  el('lon').textContent = c.longitude.toFixed(6);
  el('speed').textContent = speedKmh.toFixed(1) + ' km/h';
  el('acc').textContent = Math.round(c.accuracy) + ' m';

  try {
    const res = await fetch('/api/v1/gps/ingest', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        device_key: deviceKey,
        lat: c.latitude,
        lon: c.longitude,
        speed_kmh: speedKmh,
        heading_deg: headingDeg,
        altitude_m: c.altitude || 0,
        ignition_on: true,
        fuel_level: 0,
        engine_hours: 0,
        recorded_at: new Date().toISOString(),
      }),
    });

    if (res.ok) {
      sendCount++;
      el('count').textContent = sendCount;
      setStatus('Enviando…', 'on');
      el('msg').textContent = '';
    } else {
      const body = await res.text();
      setStatus('Error del servidor', 'err');
      el('msg').textContent = res.status + ': ' + body;
    }
  } catch (err) {
    setStatus('Sin conexión al servidor', 'err');
    el('msg').textContent = String(err);
  }
}

function handleError(err) {
  setStatus('Error de ubicación', 'err');
  el('msg').textContent = err.message;
}

el('startBtn').addEventListener('click', () => {
  if (!navigator.geolocation) {
    el('msg').textContent = 'Este navegador no soporta geolocalización.';
    return;
  }
  watchId = navigator.geolocation.watchPosition(sendPosition, handleError, {
    enableHighAccuracy: true,
    maximumAge: 2000,
    timeout: 15000,
  });
  setStatus('Buscando señal…', 'on');
  el('startBtn').style.display = 'none';
  el('stopBtn').style.display = 'block';
});

el('stopBtn').addEventListener('click', () => {
  if (watchId != null) navigator.geolocation.clearWatch(watchId);
  watchId = null;
  setStatus('Detenido');
  el('startBtn').style.display = 'block';
  el('stopBtn').style.display = 'none';
});
</script>
</body>
</html>`