package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"messenger/protocol"
	"net"
	"os"
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

	srv.storage.LogEvent("SERVER_START", "Сервер запущен на :8080")
	fmt.Println("Сервер запущен...")
	//горутина для обработки команд админа
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Println("Доступные команды сервера: /logs, /msglogs, /users, /exit")
		for scanner.Scan() {
			command := scanner.Text()
			switch command {
			case "/logs":
				logs, _ := srv.storage.GetSystemLogs(20)
				for _, l := range logs {
					fmt.Println(l)
				}
			case "/users":
				srv.mu.Lock()
				fmt.Println("В сети:", len(srv.clients))
				for name := range srv.clients {
					fmt.Println("-", name)
				}
				srv.mu.Unlock()
			case "/msglogs":
				msgLogs, _ := srv.storage.GetMessageLogs(50)
				if len(msgLogs) == 0 {
					fmt.Println("История сообщений пуста")
				} else {
					for _, l := range msgLogs {
						fmt.Println(l)
					}
				}
			//нужно доработать потом!
			case "/exit":
				os.Exit(0)
			default:
				fmt.Println("Неизвестная команда")

			}
		}
	}()

	defer srv.storage.LogEvent("SERVER_STOP", "Сервер остановлен пользователем.")
	//слушать порт
	listener, _ := net.Listen("tcp", ":8080")

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
	if s.storage.UserExists(authData.Sender) {
		if !s.storage.CheckPassword(authData.Sender, authData.Content) {
			return
		}
	} else {
		s.storage.RegisterUser(authData.Sender, authData.Content)
	}
	protocol.WritePacket(conn, protocol.TypeAuth, protocol.Message{Content: "Ok"})
	username := authData.Sender
	s.storage.LogEvent("USER_LOGIN", "Пользователь "+username+" вошел в сеть")
	//блокирование мьютекса на запись в мапу
	s.mu.Lock()
	s.clients[username] = conn
	s.mu.Unlock()
	fmt.Printf("Пользователь %s вошел в чат\n", username)
	defer func() {
		s.storage.LogEvent("USER_LOGOUT", "Пользователь "+username+" вышел из сети")
		conn.Close()
	}()
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

	}
}
