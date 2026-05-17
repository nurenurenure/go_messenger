package main

import (
	"fmt"
	"messenger/protocol"
	"strings"
	"time"
)

// formatMessageContent форматирует сообщение с учетом ответа/пересылки
func (m *model) formatMessageContent(content string) string {
	if m.forwardMsg != nil {
		fwdText := fmt.Sprintf("  >[FWD от %s]: %s", m.forwardMsg.Sender, m.forwardMsg.Content)
		if content != "" {
			return content + fwdText
		}
		return "Пересланное сообщение" + fwdText
	}

	if m.replyTo != nil && content != "" {
		preview := m.replyTo.Content
		if len(preview) > 30 {
			preview = preview[:27] + "..."
		}
		return fmt.Sprintf("%s  >[%s]: %s", content, m.replyTo.Sender, preview)
	}

	return content
}

// sendMessage отправляет сообщение в активный чат
func (m *model) sendMessage(content string) {
	if m.activeChat == "" {
		return
	}
	if strings.HasPrefix(content, "file://") {
		filePath := strings.TrimPrefix(content, "file://")
		m.handleFileCommand(fmt.Sprintf("/file %s %s", m.activeChat, filePath))
		return
	}
	content = ReplaceEmojis(content)

	finalContent := m.formatMessageContent(content)

	pMsg := protocol.Message{
		Sender:    m.username,
		Recipient: m.activeChat,
		Content:   finalContent,
		Action:    protocol.TypeChat,
		TimeStamp: time.Now().Unix(),
	}

	err := protocol.WritePacket(m.conn, protocol.TypeChat, pMsg)
	if err != nil {
		m.addMessageToChat(m.activeChat, chatMessage{
			Sender:    "Система",
			Content:   "Ошибка сети: " + err.Error(),
			Timestamp: time.Now(),
			IsMe:      false,
		})
		return
	}

	// Добавляем своё сообщение локально
	msgToSave := chatMessage{
		Sender:    m.username,
		Content:   finalContent,
		Timestamp: time.Now(),
		IsMe:      true,
	}
	m.addMessageToChat(m.activeChat, msgToSave)

	m.replyTo = nil
	m.forwardMsg = nil
	m.refreshViewport()
}

// navigateReply прокручивает историю для ответа
func (m *model) navigateReply(direction int) {
	history := m.chats[m.activeChat]
	if len(history) == 0 {
		return
	}

	// direction: -1 вверх, +1 вниз
	if direction < 0 {
		if m.replyTo == nil {
			m.selectedMsgIdx = len(history) - 1
		} else if m.selectedMsgIdx > 0 {
			m.selectedMsgIdx--
		}
		m.replyTo = &history[m.selectedMsgIdx]
	} else {
		if m.replyTo != nil {
			if m.selectedMsgIdx < len(history)-1 {
				m.selectedMsgIdx++
				m.replyTo = &history[m.selectedMsgIdx]
			} else {
				m.replyTo = nil
			}
		}
	}

	m.refreshViewport()
}

// setForwardMessage устанавливает сообщение для пересылки
func (m *model) setForwardMessage() {
	history := m.chats[m.activeChat]
	if len(history) > 0 && m.replyTo != nil {
		m.forwardMsg = m.replyTo
		m.replyTo = nil
		m.refreshViewport()
	}
}
