package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *model) refreshViewport() {
	if m.activeChat == "" {
		m.viewport.SetContent("--- Выберите чат или добавьте контакт ---")
		return
	}

	history := m.chats[m.activeChat]
	if len(history) == 0 {
		m.viewport.SetContent(fmt.Sprintf("--- Начало переписки с %s ---", m.activeChat))
		return
	}

	var lines []string
	for i, msg := range history {
		timeStr := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(msg.Timestamp.Format("15:04"))

		nameColor := "12" // Синий для других
		if msg.IsMe {
			nameColor = "2" // Зеленый для себя
		}
		nameStr := lipgloss.NewStyle().
			Foreground(lipgloss.Color(nameColor)).
			Bold(true).
			Render(msg.Sender)

		displayContent := formatQuoteDisplay(msg.Content)

		line := fmt.Sprintf("%s %s: %s", timeStr, nameStr, displayContent)

		if m.replyTo != nil && i == m.selectedMsgIdx {
			line = lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Foreground(lipgloss.Color("229")).
				Bold(true).
				Render("> " + line)
		} else {
			line = " " + line
		}

		lines = append(lines, line)
	}

	m.viewport.SetContent(strings.Join(lines, "\n"))

	if m.replyTo == nil {
		m.viewport.GotoBottom()
	}
}

// formatQuoteDisplay форматирует цитаты в сообщениях
func formatQuoteDisplay(content string) string {
	if strings.Contains(content, "  >[") {
		parts := strings.SplitN(content, "  >[", 2)
		if len(parts) == 2 {
			mainText := parts[0]
			quoteText := "  >[" + parts[1]
			quoteStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("242")).
				Italic(true)
			return mainText + quoteStyle.Render(quoteText)
		}
	}
	return content
}

// renderSidebar отрисовывает боковую панель
func (m model) renderSidebar() string {
	sidebarStyle := lipgloss.NewStyle().
		Width(20).
		Height(15).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		Padding(1)

	var sb strings.Builder
	sb.WriteString("ЧАТЫ \n\n")

	for _, c := range m.contacts {
		if c == m.activeChat {
			sb.WriteString(lipgloss.NewStyle().
				Background(lipgloss.Color("57")).
				Foreground(lipgloss.Color("255")).
				Render(c) + "\n")
		} else {
			sb.WriteString(" " + c + "\n")
		}
	}

	return sidebarStyle.Render(sb.String())
}

// renderStatusLine отображает статус пересылки
func (m model) renderStatusLine() string {
	if m.forwardMsg != nil {
		fwdStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("13")).
			Bold(true)
		return fwdStyle.Render(fmt.Sprintf(
			"ПЕРЕСЫЛКА (от %s): %s",
			m.forwardMsg.Sender,
			m.forwardMsg.Content,
		)) + "\n"
	}
	return ""
}
