package main

import (
	"messenger/protocol"
	"strings"
	"time"
)

// handleCommand обрабатывает команды и возвращает true, если это была команда
func (m *model) handleCommand(content string) bool {
	switch {
	case strings.HasPrefix(content, "/add "):
		name := strings.TrimSpace(strings.TrimPrefix(content, "/add "))
		m.addContact(name)
		return true

	case strings.HasPrefix(content, "/join "):
		groupName := strings.TrimSpace(strings.TrimPrefix(content, "/join "))
		if !strings.HasPrefix(groupName, "#") {
			groupName = "#" + groupName
		}

		if groupName != "#" {
			m.addContact(groupName)

			// Отправляем сообщение о присоединении
			joinMsg := protocol.Message{
				Sender:    m.username,
				Recipient: groupName,
				Content:   "Присоединился к группе",
				Action:    protocol.TypeChat,
				TimeStamp: time.Now().Unix(),
			}
			protocol.WritePacket(m.conn, protocol.TypeChat, joinMsg)

			// Локальное системное сообщение
			m.addMessageToChat(groupName, chatMessage{
				Sender:    m.username,
				Content:   "Вы присоединились к группе",
				Timestamp: time.Now(),
				IsMe:      true,
			})
		}
		return true

	case strings.HasPrefix(content, "/leave "):
		target := strings.TrimSpace(strings.TrimPrefix(content, "/leave "))
		m.removeChat(target)

		// Системное сообщение серверу
		leaveMsg := protocol.Message{
			Sender:    m.username,
			Recipient: target,
			Action:    protocol.TypeSystem,
			Content:   "LEAVE",
		}
		protocol.WritePacket(m.conn, protocol.TypeSystem, leaveMsg)
		return true
	}

	return false
}
