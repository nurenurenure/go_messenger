package main

import (
	"messenger/protocol"
	"net"
	"sync"
)

type Server struct {
	clients map[string]net.Conn            // ключ - никнейм, значение - соединение
	groups  map[string]map[string]net.Conn // группы пользователей
	mu      sync.Mutex
	storage *Storage
}

func NewServer() *Server {
	return &Server{
		clients: make(map[string]net.Conn),
		groups:  make(map[string]map[string]net.Conn),
	}
}

// addClient добавляет клиента в общий список
func (s *Server) addClient(username string, conn net.Conn) {
	s.mu.Lock()
	s.clients[username] = conn
	s.mu.Unlock()
}

// removeClient удаляет клиента из всех списков
func (s *Server) removeClient(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Удалить из общего списка
	delete(s.clients, username)

	// Удалить из всех групп
	for groupName := range s.groups {
		delete(s.groups[groupName], username)
		if len(s.groups[groupName]) == 0 {
			delete(s.groups, groupName)
		}
	}
}

// addToGroup добавляет пользователя в группу
func (s *Server) addToGroup(groupName, username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.groups[groupName]; !exists {
		s.groups[groupName] = make(map[string]net.Conn)
	}

	if _, inGroup := s.groups[groupName][username]; !inGroup {
		if conn, ok := s.clients[username]; ok {
			s.groups[groupName][username] = conn
		}
	}
}

// broadcastToGroup рассылает сообщение всем участникам группы
func (s *Server) broadcastToGroup(groupName, sender string, msg protocol.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	group, exists := s.groups[groupName]
	if !exists {
		return
	}

	for name, conn := range group {
		if name != sender {
			protocol.WritePacket(conn, protocol.TypeChat, msg)
		}
	}
}

// sendToUser отправляет сообщение конкретному пользователю
func (s *Server) sendToUser(username string, msg protocol.Message) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	conn, ok := s.clients[username]
	if !ok {
		return false
	}

	protocol.WritePacket(conn, protocol.TypeChat, msg)
	return true
}

// getOnlineUsers возвращает список пользователей онлайн
func (s *Server) getOnlineUsers() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	users := make([]string, 0, len(s.clients))
	for name := range s.clients {
		users = append(users, name)
	}
	return users
}

// getOnlineCount возвращает количество пользователей онлайн
func (s *Server) getOnlineCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.clients)
}
