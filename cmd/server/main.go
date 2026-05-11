package main

import (
	"encoding/json"
	"fmt"
	"messenger/protocol"
	"net"
	"sync"

	_ "modernc.org/sqlite"
)

type Server struct {
	//ключ - никнейм, значение - соединение
	clients map[string]net.Conn
	//горутины будут одновременно записывать данные в мапу
	mu      sync.Mutex
	storage *Storage
}

// конструктор сервера
func NewServer() *Server {
	return &Server{
		clients: make(map[string]net.Conn),
	}
}

func main() {
	//инициализировать сервер
	srv := NewServer()
	//инициализировать базу
	srv.storage = InitStorage("./messenger.db")
	//слушать порт
	listener, _ := net.Listen("tcp", ":8080")
	fmt.Println("Сервер запущен на :8080")

	for {
		conn, _ := listener.Accept()
		// горутина для каждого клиента
		go srv.handleClient(conn)
	}
}

func (s *Server) handleClient(conn net.Conn) {
	defer conn.Close()
	//первым пакетом должен быть пакет авторизации
	msgType, payload, err := protocol.ReadPacket(conn)
	if err != nil || msgType != protocol.TypeAuth {
		return
	}

	var authData protocol.Message
	json.Unmarshal(payload, &authData)
	username := authData.Sender
	//блокирование мьютекса на запись в мапу
	s.mu.Lock()
	s.clients[username] = conn
	s.mu.Unlock()
	fmt.Printf("Пользователь %s вошел в чат\n", username)
	//получить историю сообщений
	history, err := s.storage.GetHistory(username)

	if err != nil {
		fmt.Println("Ошибка получения истории:", err)
	} else {
		for _, msg := range history {
			protocol.WritePacket(conn, protocol.TypeChat, msg)
		}
	}
	for {
		msgType, payload, err := protocol.ReadPacket(conn)
		//удаление записи при обрыве соединения
		if err != nil {
			s.mu.Lock()
			delete(s.clients, username)
			s.mu.Unlock()
			return
		}

		if msgType == protocol.TypeChat {
			s.routeMessage(payload)
		}
	}
}

func (s *Server) routeMessage(payload []byte) {
	var msg protocol.Message
	json.Unmarshal(payload, &msg)

	//сохранить в БД
	err := s.storage.SaveMessage(msg)
	if err != nil {
		fmt.Println("Ошибка сохранения в БД:", err)
	}

	//чтение из мапы
	s.mu.Lock()
	recipientConn, ok := s.clients[msg.Recipient]
	s.mu.Unlock()
	if ok {
		//отправка сообщения получателю
		protocol.WritePacket(recipientConn, protocol.TypeChat, msg)

	} else {
		fmt.Printf("Пользователь %s не в сети\n", msg.Recipient)
	}
}
