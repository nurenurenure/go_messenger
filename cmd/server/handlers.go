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

	// Аутентификация
	username, err := s.authenticate(conn)
	if err != nil {
		return
	}

	// Регистрация клиента
	s.addClient(username, conn)
	defer s.cleanupClient(username, conn)

	s.storage.LogEvent("USER_LOGIN", "Пользователь "+username+" вошел в сеть")
	fmt.Printf("Пользователь %s вошел в чат\n", username)

	// Отправка истории сообщений
	s.sendHistory(conn, username)

	// Основной цикл обработки сообщений
	s.messageLoop(conn, username)
}

// authenticate выполняет аутентификацию пользователя
func (s *Server) authenticate(conn net.Conn) (string, error) {
	msgType, payload, err := protocol.ReadPacket(conn)
	if err != nil || msgType != protocol.TypeAuth {
		return "", fmt.Errorf("ошибка авторизации")
	}

	var authData protocol.Message
	json.Unmarshal(payload, &authData)

	// Проверка пользователя
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

// sendHistory отправляет историю сообщений пользователю
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

// messageLoop обрабатывает входящие сообщения
func (s *Server) messageLoop(conn net.Conn, username string) {
	for {
		msgType, payload, err := protocol.ReadPacket(conn)
		if err != nil {
			return // соединение разорвано
		}

		switch msgType {
		case protocol.TypeChat:
			s.handleChatMessage(payload)
		case protocol.TypeSystem:
			s.handleSystemMessage(payload, username)
		}
	}
}

// handleChatMessage обрабатывает чат-сообщение
func (s *Server) handleChatMessage(payload []byte) {
	var msg protocol.Message
	json.Unmarshal(payload, &msg)

	// Сохраняем в БД
	if err := s.storage.SaveMessage(msg); err != nil {
		fmt.Println("Ошибка сохранения в БД:", err)
	}

	// Маршрутизация сообщения
	if strings.HasPrefix(msg.Recipient, "#") {
		// Групповое сообщение
		s.addToGroup(msg.Recipient, msg.Sender)
		s.broadcastToGroup(msg.Recipient, msg.Sender, msg)
	} else {
		// Личное сообщение
		s.sendToUser(msg.Recipient, msg)
	}
}

// handleSystemMessage обрабатывает системное сообщение
func (s *Server) handleSystemMessage(payload []byte, username string) {
	var sysMsg protocol.Message
	json.Unmarshal(payload, &sysMsg)

	if sysMsg.Content == "LEAVE" {
		s.removeClient(username)
		s.storage.RemoveUserFromGroupHistory(username, sysMsg.Recipient)
		fmt.Printf("Пользователь %s покинул чат %s\n", username, sysMsg.Recipient)
	}
}

// cleanupClient очищает данные клиента при отключении
func (s *Server) cleanupClient(username string, conn net.Conn) {
	s.storage.LogEvent("USER_LOGOUT", "Пользователь "+username+" вышел из сети")
	s.removeClient(username)
	conn.Close()
	fmt.Printf("Пользователь %s покинул чат\n", username)
}
