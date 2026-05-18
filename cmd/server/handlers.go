package main

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"messenger/protocol"
)

func (s *Server) handleClient(conn net.Conn) {
	defer conn.Close()

	username, err := s.authenticate(conn)
	if err != nil {
		return
	}
	if s.shuttingDown {
		return
	}

	s.addClient(username, conn)
	defer s.cleanupClient(username, conn)

	s.storage.LogEvent("USER_LOGIN", "Пользователь "+username+" вошел в сеть")
	fmt.Printf("Пользователь %s вошел в чат\n", username)

	s.sendHistory(conn, username)
	s.messageLoop(conn, username)
}

func (s *Server) authenticate(conn net.Conn) (string, error) {
	msgType, payload, err := protocol.ReadPacket(conn)
	if err != nil || msgType != protocol.TypeAuth {
		return "", fmt.Errorf("ошибка авторизации")
	}

	var authData protocol.Message
	json.Unmarshal(payload, &authData)

	if s.storage.UserExists(authData.Sender) {
		if !s.storage.CheckPassword(authData.Sender, authData.Content) {
			return "", fmt.Errorf("неверный пароль")
		}
	} else {
		s.storage.RegisterUser(authData.Sender, authData.Content)
	}

	protocol.WritePacket(conn, protocol.TypeAuth, protocol.Message{Content: "Ok"})
	return authData.Sender, nil
}

func (s *Server) sendHistory(conn net.Conn, username string) {
	history, err := s.storage.GetHistory(username)
	if err != nil {
		fmt.Println("Ошибка получения истории:", err)
		return
	}

	for _, msg := range history {
		protocol.WritePacket(conn, protocol.TypeChat, msg)
	}
}

func (s *Server) messageLoop(conn net.Conn, username string) {
	for {
		msgType, payload, err := protocol.ReadPacket(conn)
		if err != nil {
			return
		}

		switch msgType {
		case protocol.TypeChat:
			s.handleChatMessage(payload)
		case protocol.TypeSystem:
			s.handleSystemMessage(payload, username)
		case protocol.TypeFileRequest,
			protocol.TypeFileAccept,
			protocol.TypeFileReject,
			protocol.TypeFileHeader,
			protocol.TypeFileChunk,
			protocol.TypeFileComplete:
			s.handleFileTransfer(msgType, payload, username)
		}
	}
}

func (s *Server) handleChatMessage(payload []byte) {
	var msg protocol.Message
	json.Unmarshal(payload, &msg)

	if err := s.storage.SaveMessage(msg); err != nil {
		fmt.Println("Ошибка сохранения в БД:", err)
	}

	if strings.HasPrefix(msg.Recipient, "#") {
		s.addToGroup(msg.Recipient, msg.Sender)
		s.broadcastToGroup(msg.Recipient, msg.Sender, msg)
	} else {
		s.sendToUser(msg.Recipient, msg)
	}
}

func (s *Server) handleSystemMessage(payload []byte, username string) {
	var sysMsg protocol.Message
	json.Unmarshal(payload, &sysMsg)

	if sysMsg.Content == "LEAVE" {
		s.removeClient(username)
		s.storage.RemoveUserFromGroupHistory(username, sysMsg.Recipient)
		fmt.Printf("Пользователь %s покинул чат %s\n", username, sysMsg.Recipient)
	}
}

func (s *Server) cleanupClient(username string, conn net.Conn) {
	s.storage.LogEvent("USER_LOGOUT", "Пользователь "+username+" вышел из сети")
	s.removeClient(username)
	conn.Close()
	fmt.Printf("Пользователь %s покинул чат\n", username)
}
