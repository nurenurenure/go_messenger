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

// Структура сообщения специально для интерфейса клиента
type chatMessage struct {
	Sender    string
	Content   string
	Timestamp time.Time
	IsMe      bool
}

type model struct {
	viewport   viewport.Model
	textarea   textarea.Model
	chats      map[string][]chatMessage
	contacts   []string
	activeChat string
	username   string
	conn       net.Conn
}

// Специальный тип для обработки сообщений от сервера в Bubble Tea
type serverMsg protocol.Message

func InitialModel(conn net.Conn, username string) model {
	ta := textarea.New()
	ta.Placeholder = "Введите сообщение или /add [ник]..."
	ta.Focus()
	ta.CharLimit = 280
	ta.SetHeight(3)

	vp := viewport.New(55, 15)
	vp.SetContent("--- Выберите чат (TAB) или добавьте контакт через /add ---")

	return model{
		textarea:   ta,
		viewport:   vp,
		conn:       conn,
		username:   username,
		chats:      make(map[string][]chatMessage),
		contacts:   []string{},
		activeChat: "",
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyTab:
			if len(m.contacts) > 0 {
				currentIndex := -1
				for i, contact := range m.contacts {
					if contact == m.activeChat {
						currentIndex = i
						break
					}
				}
				nextIndex := (currentIndex + 1) % len(m.contacts)
				m.activeChat = m.contacts[nextIndex]
				m.refreshViewPoint()
			}
			return m, nil

		case tea.KeyEnter:
			//TrimSpace убирает \n, который textarea успевает вставить
			content := strings.TrimSpace(m.textarea.Value())
			if content == "" {
				m.textarea.Reset()
				return m, nil
			}

			// Обработка команды /add
			if strings.HasPrefix(content, "/add ") {
				name := strings.TrimSpace(strings.TrimPrefix(content, "/add "))
				if name != "" && name != m.username {
					if _, exists := m.chats[name]; !exists {
						m.chats[name] = []chatMessage{}
						m.contacts = append(m.contacts, name)
					}
					m.activeChat = name
					m.refreshViewPoint()
				}
				m.textarea.Reset()
				return m, nil // Выходим, чтобы не отправлять команду как текст
			}

			// Отправка обычного сообщения
			if m.activeChat != "" {
				pMsg := protocol.Message{
					Sender:    m.username,
					Recipient: m.activeChat,
					Content:   content,
					Action:    protocol.TypeChat,
					TimeStamp: time.Now().Unix(),
				}

				err := protocol.WritePacket(m.conn, protocol.TypeChat, pMsg)
				if err != nil {
					m.chats[m.activeChat] = append(m.chats[m.activeChat], chatMessage{
						Sender:    "Система",
						Content:   "Ошибка сети: " + err.Error(),
						Timestamp: time.Now(),
						IsMe:      false,
					})
				} else {
					m.chats[m.activeChat] = append(m.chats[m.activeChat], chatMessage{
						Sender:    m.username,
						Content:   content,
						Timestamp: time.Now(),
						IsMe:      true,
					})
				}
				m.refreshViewPoint()
			}
			m.textarea.Reset()
			return m, nil
		}

	case serverMsg:
		sender := msg.Sender
		newMsg := chatMessage{
			Sender:    sender,
			Content:   msg.Content,
			Timestamp: time.Unix(msg.TimeStamp, 0),
			IsMe:      false,
		}

		if _, exists := m.chats[sender]; !exists {
			m.chats[sender] = []chatMessage{}
			m.contacts = append(m.contacts, sender)
			if m.activeChat == "" {
				m.activeChat = sender
			}
		}
		m.chats[sender] = append(m.chats[sender], newMsg)
		if m.activeChat == sender {
			m.refreshViewPoint()
		}
		return m, nil
	}

	// Обновляем внутренние компоненты (только если не вышли по Enter выше)
	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m *model) refreshViewPoint() {
	if m.activeChat == "" {
		m.viewport.SetContent("--- Выберите чат или добавьте контакт ---")
		return
	}

	var lines []string
	history := m.chats[m.activeChat]

	if len(history) == 0 {
		m.viewport.SetContent(fmt.Sprintf("--- Начало переписки с %s ---", m.activeChat))
		return
	}

	for _, msg := range history {
		t := msg.Timestamp.Format("15:04")
		timeStr := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(t)

		nameColor := "12" // Синий для других
		if msg.IsMe {
			nameColor = "2" // Зеленый для себя
		}
		nameStr := lipgloss.NewStyle().Foreground(lipgloss.Color(nameColor)).Bold(true).Render(msg.Sender)
		lines = append(lines, fmt.Sprintf("%s %s: %s", timeStr, nameStr, msg.Content))
	}
	m.viewport.SetContent(strings.Join(lines, "\n"))
	m.viewport.GotoBottom()
}

func (m model) View() string {
	sidebarStyle := lipgloss.NewStyle().
		Width(20).
		Height(15).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		Padding(1)

	var sb strings.Builder
	sb.WriteString("ЧАТЫ \n\n")
	for _, c := range m.contacts {
		if c == m.activeChat {
			sb.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("57")).Foreground(lipgloss.Color("255")).Render(c) + "\n")
		} else {
			sb.WriteString(" " + c + "\n")
		}
	}

	var centerView string
	if m.activeChat == "" {
		centerView = "\n\n  У Вас пока нет активных диалогов.\n  Введите /add [имя], чтобы начать."
	} else {
		centerView = m.viewport.View()
	}

	mainLayout := lipgloss.JoinHorizontal(lipgloss.Top, sidebarStyle.Render(sb.String()), centerView)

	return fmt.Sprintf(
		"Вы вошли как: %s\n\n%s\n\n%s\n%s",
		m.username,
		mainLayout,
		m.textarea.View(),
		"TAB - сменить чат | /add имя - начать чат | Ctrl+C - выход",
	)
}

func waitForMessages(conn net.Conn, p *tea.Program) {
	for {
		msgType, payload, err := protocol.ReadPacket(conn)
		if err != nil {
			break
		}
		if msgType == protocol.TypeChat {
			var incoming protocol.Message
			if err := json.Unmarshal(payload, &incoming); err == nil {
				p.Send(serverMsg(incoming))
			}
		}
	}
}

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Ошибка подключения к серверу:", err)
		return
	}

	fmt.Print("Введите Ваш никнейм: ")
	var username string
	fmt.Scanln(&username)
	fmt.Print("Введите Ваш пароль: ")
	var password string
	fmt.Scanln(&password)

	// Авторизация
	authMsg := protocol.Message{
		Sender:  username,
		Content: password,
		Action:  protocol.TypeAuth,
	}
	protocol.WritePacket(conn, protocol.TypeAuth, authMsg)

	msgType, _, err := protocol.ReadPacket(conn)
	if err != nil || msgType != protocol.TypeAuth {
		fmt.Println("Ошибка авторизации. Проверьте логин/пароль.")
		return
	}

	// Запуск интерфейса
	p := tea.NewProgram(InitialModel(conn, username), tea.WithAltScreen())

	go waitForMessages(conn, p)

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
