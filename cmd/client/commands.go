package main

import (
	"fmt"
	"messenger/protocol"
	"path/filepath"
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

	case strings.HasPrefix(content, "/file "):
		m.handleFileCommand(content)
		return true
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

func (m *model) handleFileCommand(content string) {
	parts := strings.SplitN(content, " ", 3)
	if len(parts) < 3 {
		m.addSystemMessage("Использование: /file [ник] [путь к файлу]")
		return
	}

	recipient := parts[1]
	filePath := parts[2]

	go func() {
		m.addSystemMessage(fmt.Sprintf("Отправка файла %s для %s...", filepath.Base(filePath), recipient))

		header, err := m.fileSender.SendFileRequest(recipient, filePath)
		if err != nil {
			m.addSystemMessage(fmt.Sprintf("Ошибка: %v", err))
			return
		}

		// Ждем подтверждения
		time.Sleep(1 * time.Second)

		err = m.fileSender.SendFile(*header, filePath, nil)
		if err != nil {
			m.addSystemMessage(fmt.Sprintf("Ошибка отправки: %v", err))
			return
		}

		m.addSystemMessage(fmt.Sprintf("Файл %s отправлен", header.FileName))
	}()
}

func (m *model) addSystemMessage(text string) {
	m.addMessageToChat(m.activeChat, chatMessage{
		Sender:    "Система",
		Content:   text,
		Timestamp: time.Now(),
		IsMe:      false,
	})
}
