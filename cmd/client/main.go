package main

import (
	"encoding/json"
	"fmt"
	"log"
	"messenger/protocol"
	"net"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	viewport viewport.Model //окно с текстом
	textarea textarea.Model //поле для ввода
	messages []string       //локальный кэш
	conn     net.Conn       //соединение
	username string
	err      error
}

// спец тип для интерфейса
type serverMsg protocol.Message

// инициализация
func InitialModel(conn net.Conn, username string) model {
	ta := textarea.New()
	ta.Placeholder = "Кому:сообщение..."
	ta.Focus()
	ta.CharLimit = 280
	ta.SetHeight(3)
	vp := viewport.New(80, 15)
	vp.SetContent("--- Соединение установлено. Добро пожаловать! ---\n")

	return model{
		textarea: ta,
		viewport: vp,
		conn:     conn,
		username: username,
		messages: []string{},
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink //моргающий курсор
}

// обновление
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	//обработка типов событий
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			input := m.textarea.Value()
			if input == "" {
				return m, nil
			}

			parts := strings.SplitN(input, ":", 2)
			if len(parts) < 2 {
				m.messages = append(m.messages, lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Ошибка! Формат 'Имя: текст'"))
				m.viewport.SetContent(strings.Join(m.messages, "\n"))
				m.textarea.Reset()
				return m, nil
			}

			pMsg := protocol.Message{
				Sender:    m.username,
				Recipient: strings.TrimSpace(parts[0]),
				Content:   strings.TrimSpace(parts[1]),
				Action:    protocol.TypeChat,
				TimeStamp: time.Now().Unix(),
			}
			protocol.WritePacket(m.conn, 2, pMsg)

			m.messages = append(m.messages, fmt.Sprintf("Я -> %s: %s", pMsg.Recipient, pMsg.Content))
			m.viewport.SetContent(strings.Join(m.messages, "\n"))
			m.viewport.GotoBottom()
			m.textarea.Reset()
		}

	case serverMsg:
		newMsg := fmt.Sprintf("[%s]: %s", msg.Sender, msg.Content)
		m.messages = append(m.messages, newMsg)
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()
	}
	return m, tea.Batch(tiCmd, vpCmd)
}

func (m model) View() string {
	return fmt.Sprintf("Чат: %s\n\n%s\n\n%s\n%s", m.username,
		m.viewport.View(),
		m.textarea.View(),
		lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Ctrl+C - выход | Enter - отправить"),
	)
}

// горутина для чтения из сети
func waitForMessages(conn net.Conn, p *tea.Program) {
	for {
		msgType, payload, err := protocol.ReadPacket(conn)
		if err != nil {
			break
		}
		if msgType == 2 {
			var incoming protocol.Message
			json.Unmarshal(payload, &incoming)

			p.Send(serverMsg(incoming))
		}
	}
}

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Не удалось подключиться к серверу:", err)
		return
	}

	fmt.Print("Введите Ваш никнейм: ")
	var username string
	fmt.Scanln(&username)
	fmt.Print("Введите Ваш пароль: ")
	var password string
	fmt.Scanln(&password)
	authMsg := protocol.Message{
		Sender:  username,
		Content: password,
		Action:  protocol.TypeAuth,
	}
	protocol.WritePacket(conn, protocol.TypeAuth, authMsg)

	msgType, _, err := protocol.ReadPacket(conn)
	if err != nil || msgType != protocol.TypeAuth {
		fmt.Println("Ошибка авторизации.")
		return
	}

	p := tea.NewProgram(InitialModel(conn, username), tea.WithAltScreen())

	go waitForMessages(conn, p)
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}

}
