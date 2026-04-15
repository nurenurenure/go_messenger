package server

import (
	"log"
	"messenger/internal/protocol"
	"net"
	"sync"
)

// Hub управляет всеми активными соединениями
type Hub struct {
	// mu защищает карту clients от одновременной записи из разных горутин
	mu      sync.RWMutex
	clients map[string]net.Conn
}

// NewHub создает новый экземпляр хаба
func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]net.Conn),
	}
}

// Register добавляет пользователя в список онлайн
func (h *Hub) Register(username string, conn net.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[username] = conn
	log.Printf("Пользователь %s добавлен в Hub", username)
}

// Unregister удаляет пользователя (вызывается при EOF или ошибке)
func (h *Hub) Unregister(username string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[username]; ok {
		delete(h.clients, username)
		log.Printf("Пользователь %s удален из Hub", username)
	}
}

// Broadcast отправляет сообщение ВСЕМ пользователям, кроме отправителя (опционально)
func (h *Hub) Broadcast(msgType uint16, payload []byte) {
	h.mu.RLock() // Блокируем только на чтение
	defer h.mu.RUnlock()

	for name, conn := range h.clients {
		err := protocol.WritePacket(conn, msgType, payload)
		if err != nil {
			log.Printf("Ошибка отправки пользователю %s: %v", name, err)
		}
	}
}
