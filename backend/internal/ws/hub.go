package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub mantiene la lista de clientes conectados y reparte los
// mensajes de posición a todos ellos (broadcast).
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]bool
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]bool),
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// En producción: restringir el origen a tu dominio del frontend.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeHTTP convierte la conexión HTTP en una conexión WebSocket
// y la registra en el hub.
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws: error al hacer upgrade: %v", err)
		return
	}

	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	log.Printf("ws: cliente conectado (%d activos)", len(h.clients))

	// Mantenemos la conexión abierta leyendo (y descartando) mensajes
	// del cliente hasta que se desconecte.
	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
			conn.Close()
			log.Printf("ws: cliente desconectado (%d activos)", len(h.clients))
		}()

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}

// Broadcast envía un objeto (serializado en JSON) a todos los
// clientes conectados actualmente.
func (h *Hub) Broadcast(payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ws: error serializando payload: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("ws: error enviando a cliente: %v", err)
		}
	}
}
