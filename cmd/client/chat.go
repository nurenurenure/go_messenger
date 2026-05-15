package main

import (
	"messenger/protocol"
	"strings"
)

// getTargetChat определяет ключ чата для сообщения
func getTargetChat(msg protocol.Message, myUsername string) string {
	if strings.HasPrefix(msg.Recipient, "#") {
		return msg.Recipient // группа
	}
	// личные сообщения
	if msg.Sender == myUsername {
		return msg.Recipient
	}
	return msg.Sender
}

// addMessageToChat добавляет сообщение в чат и сохраняет в БД
func (m *model) addMessageToChat(chatName string, msg chatMessage) {
	if _, exists := m.chats[chatName]; !exists {
		m.chats[chatName] = []chatMessage{}
		m.contacts = append(m.contacts, chatName)
		if m.activeChat == "" {
			m.activeChat = chatName
		}
	}

	m.chats[chatName] = append(m.chats[chatName], msg)

	if m.db != nil {
		saveToDB(m.db, chatName, msg)
	}

	if m.activeChat == chatName {
		m.refreshViewport()
	}
}

// addContact добавляет новый контакт
func (m *model) addContact(name string) bool {
	if name == "" || name == m.username {
		return false
	}

	if _, exists := m.chats[name]; exists {
		return false
	}

	m.chats[name] = loadHistory(m.db, name)
	m.contacts = append(m.contacts, name)
	m.activeChat = name
	m.refreshViewport()
	return true
}

// removeChat удаляет чат и контакт
func (m *model) removeChat(target string) {
	delete(m.chats, target)

	newContacts := []string{}
	for _, c := range m.contacts {
		if c != target {
			newContacts = append(newContacts, c)
		}
	}
	m.contacts = newContacts

	// Переключаем активный чат
	if m.activeChat == target {
		m.activeChat = ""
		if len(m.contacts) > 0 {
			m.activeChat = m.contacts[0]
		}
	}
}

// switchToNextContact переключает на следующий контакт
func (m *model) switchToNextContact() {
	if len(m.contacts) == 0 {
		return
	}

	currentIndex := -1
	for i, contact := range m.contacts {
		if contact == m.activeChat {
			currentIndex = i
			break
		}
	}

	nextIndex := (currentIndex + 1) % len(m.contacts)
	m.activeChat = m.contacts[nextIndex]
	m.refreshViewport()
}
