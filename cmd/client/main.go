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
	viewport       viewport.Model
	textarea       textarea.Model
	chats          map[string][]chatMessage
	contacts       []string
	activeChat     string
	username       string
	conn           net.Conn
	replyTo        *chatMessage
	selectedMsgIdx int
	forwardMsg     *chatMessage
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
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyUp:
			history := m.chats[m.activeChat]
			if len(history) > 0 {
				if m.replyTo == nil {
					m.selectedMsgIdx = len(history) - 1
				} else if m.selectedMsgIdx > 0 {
					m.selectedMsgIdx--
				}
				m.replyTo = &history[m.selectedMsgIdx]
				m.refreshViewPoint()
			}
			return m, nil
		case tea.KeyDown:
			history := m.chats[m.activeChat]
			if m.replyTo != nil {
				if m.selectedMsgIdx < len(history)-1 {
					m.selectedMsgIdx++
					m.replyTo = &history[m.selectedMsgIdx]
				} else {
					m.replyTo = nil
				}
				m.refreshViewPoint()
			}
			return m, nil
		case tea.KeyRight:
			history := m.chats[m.activeChat]
			if len(history) > 0 && m.replyTo != nil {
				//выбор сообщения для пересылки
				m.forwardMsg = m.replyTo

				//сброс режима выбора, но forwardMsg остается в памяти
				m.replyTo = nil
				m.refreshViewPoint()
			}
			return m, nil

		case tea.KeyEscape:
			if m.replyTo != nil {
				m.replyTo = nil
				return m, nil
			}

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
			if content == "" && m.forwardMsg == nil {
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

			//обработка команды /join
			if strings.HasPrefix(content, "/join ") {
				groupname := strings.TrimSpace(strings.TrimPrefix(content, "/join"))

				//если пользователь не написал решетку, написать
				if !strings.HasPrefix(groupname, "#") {
					groupname = "#" + groupname
				}

				if groupname != "#" {
					if _, exists := m.chats[groupname]; !exists {
						m.chats[groupname] = []chatMessage{}
						m.contacts = append(m.contacts, groupname)
					}
					m.activeChat = groupname
					m.refreshViewPoint()

					joinMsg := protocol.Message{
						Sender:    m.username,
						Recipient: groupname,
						Content:   "Присоединился к группе",
						Action:    protocol.TypeChat,
						TimeStamp: time.Now().Unix(),
					}
					protocol.WritePacket(m.conn, protocol.TypeChat, joinMsg)

					//локальное сообщение пользователю
					m.chats[groupname] = append(m.chats[groupname], chatMessage{
						Sender:    m.username,
						Content:   "Вы присоединились к группе",
						Timestamp: time.Now(),
						IsMe:      true,
					})
					m.refreshViewPoint()
				}
				m.textarea.Reset()
				return m, nil
			}

			//обработка команды /leave
			if strings.HasPrefix(content, "/leave ") {
				target := strings.TrimSpace(strings.TrimPrefix(content, "/leave "))

				delete(m.chats, target)
				newContacts := []string{}
				for _, c := range m.contacts {
					if c != target {
						newContacts = append(newContacts, c)
					}
				}
				m.contacts = newContacts

				//если пользователь вышел из активного чата, сбросить его
				if m.activeChat == target {
					m.activeChat = ""
					if len(m.contacts) > 0 {
						m.activeChat = m.contacts[0]
					} else {
						m.activeChat = ""
					}
				}
				m.textarea.Reset()
				m.refreshViewPoint()
				//системное сообщение серверу
				leaveMsg := protocol.Message{
					Sender:    m.username,
					Recipient: target,
					Action:    protocol.TypeSystem,
					Content:   "LEAVE",
				}
				protocol.WritePacket(m.conn, protocol.TypeSystem, leaveMsg)
				return m, nil
			}

			// Отправка обычного сообщения
			if m.activeChat != "" {
				finalContent := content

				if m.forwardMsg != nil {
					//формат пересылки
					fwdText := fmt.Sprintf("  >[FWD от %s]: %s", m.forwardMsg.Sender, m.forwardMsg.Content)
					if content != "" {
						finalContent = content + fwdText
					} else {
						finalContent = "Пересланное сообщение" + fwdText
					}
				} else if m.replyTo != nil {

					preview := m.replyTo.Content
					if len(preview) > 30 {
						preview = preview[:27] + "..."
					}
					finalContent = fmt.Sprintf("%s  >[%s]: %s", content, m.replyTo.Sender, preview)

				}
				pMsg := protocol.Message{
					Sender:    m.username,
					Recipient: m.activeChat,
					Content:   finalContent,
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
						Content:   finalContent,
						Timestamp: time.Now(),
						IsMe:      true,
					})
				}

				m.replyTo = nil
				m.refreshViewPoint()
			}
			m.textarea.Reset()
			return m, nil
		}

	case serverMsg:
		var targetChat string
		//если личка,ключ - имя отправителя, если группа - название группы
		if strings.HasPrefix(msg.Recipient, "#") {
			targetChat = msg.Recipient
		} else {
			if msg.Sender == m.username {
				targetChat = msg.Recipient
			} else {
				targetChat = msg.Sender
			}
		}

		if _, exists := m.chats[targetChat]; !exists {
			m.chats[targetChat] = []chatMessage{}
			m.contacts = append(m.contacts, targetChat)
			if m.activeChat == "" {
				m.activeChat = targetChat
			}
		}
		m.chats[targetChat] = append(m.chats[targetChat], chatMessage{
			Sender:    msg.Sender,
			Content:   msg.Content,
			Timestamp: time.Unix(msg.TimeStamp, 0),
			IsMe:      (msg.Sender == m.username),
		})
		if m.activeChat == targetChat {
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

	for i, msg := range history {
		t := msg.Timestamp.Format("15:04")
		timeStr := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(t)

		nameColor := "12" // Синий для других
		if msg.IsMe {
			nameColor = "2" // Зеленый для себя
		}
		nameStr := lipgloss.NewStyle().Foreground(lipgloss.Color(nameColor)).Bold(true).Render(msg.Sender)
		displayContent := msg.Content

		//ищем разделители читаты
		if strings.Contains(msg.Content, "  >[") {
			parts := strings.SplitN(msg.Content, "  >[", 2)
			if len(parts) == 2 {
				mainText := parts[0]
				quoteText := "  >[" + parts[1]

				quoteStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Italic(true)
				displayContent = mainText + quoteStyle.Render(quoteText)
			}
		}
		line := fmt.Sprintf("%s %s: %s", timeStr, nameStr, displayContent)
		if m.replyTo != nil && i == m.selectedMsgIdx {
			style := lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Foreground(lipgloss.Color("229")).
				Bold(true)
			lines = append(lines, style.Render("> "+line))
		} else {
			lines = append(lines, " "+line)
		}
	}
	m.viewport.SetContent(strings.Join(lines, "\n"))
	//
	if m.replyTo == nil {
		m.viewport.GotoBottom()

	}
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

	statusLine := ""
	if m.forwardMsg != nil {
		fwdStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
		statusLine = fwdStyle.Render(fmt.Sprintf("ПЕРЕСЫЛКА (от %s): %s", m.forwardMsg.Sender, m.forwardMsg.Content)) + "\n"
	}

	return fmt.Sprintf(
		"Вы вошли как: %s\n\n%s\n\n%s%s\n%s",
		m.username,
		mainLayout,
		statusLine,
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
