package main

import (
	"net"
	"sync"

	"messenger/protocol"
)

type Server struct {
	clients       map[string]net.Conn
	groups        map[string]map[string]net.Conn
	mu            sync.Mutex
	storage       *Storage
	fileTransfers *FileTransferManager // ДОБАВИТЬ
}

func NewServer() *Server {
	return &Server{
		clients:       make(map[string]net.Conn),
		groups:        make(map[string]map[string]net.Conn),
		fileTransfers: NewFileTransferManager(), // ДОБАВИТЬ
	}
}

func (s *Server) addClient(username string, conn net.Conn) {
	s.mu.Lock()
	s.clients[username] = conn
	s.mu.Unlock()
}

func (s *Server) removeClient(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.clients, username)

	for groupName := range s.groups {
		delete(s.groups[groupName], username)
		if len(s.groups[groupName]) == 0 {
			delete(s.groups, groupName)
		}
	}
}

func (s *Server) getClient(username string) (net.Conn, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	conn, ok := s.clients[username]
	return conn, ok
}

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

func (s *Server) getOnlineUsers() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	users := make([]string, 0, len(s.clients))
	for name := range s.clients {
		users = append(users, name)
	}
	return users
}

func (s *Server) getOnlineCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.clients)
}
