package main

import (
	"fmt"
	"messenger/protocol"
	"net"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dgraph-io/badger/v4"
)

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
	db             *badger.DB
	fileReceiver   *FileReceiver
	fileSender     *FileSender
	fileProgress   float64
}

type serverMsg protocol.Message

func InitialModel(conn net.Conn, username string, db *badger.DB) *model {
	ta := textarea.New()
	ta.Placeholder = "Введите сообщение, /add [ник] или /file [ник] [путь]..."
	ta.Focus()
	ta.CharLimit = 280
	ta.SetHeight(3)

	vp := viewport.New(55, 15)
	vp.SetContent("--- Выберите чат (TAB) или добавьте контакт через /add ---")

	m := &model{
		textarea:     ta,
		viewport:     vp,
		conn:         conn,
		username:     username,
		chats:        make(map[string][]chatMessage),
		contacts:     []string{},
		activeChat:   "",
		db:           db,
		fileReceiver: NewFileReceiver(conn, username),
		fileSender:   NewFileSender(conn, username),
	}

	m.contacts, m.chats = loadAllContacts(db)
	if len(m.contacts) > 0 && m.activeChat == "" {
		m.activeChat = m.contacts[0]
		m.refreshViewport()
	}

	// Настройка callback для получения файлов
	m.fileReceiver.SetOnComplete(func(fileName string, filePath string) {
		// Добавляем сообщение в чат о полученном файле
	})

	return m
}

func (m *model) Init() tea.Cmd {
	return textarea.Blink
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if isSpecialKey(msg) {
			return m.handleKeyPress(msg)
		}

	case serverMsg:
		protocolMsg := protocol.Message(msg)

		// Проверяем системные сообщения
		if protocolMsg.Action == protocol.TypeSystem {
			if strings.Contains(protocolMsg.Content, "Сервер завершает работу") {
				// Показываем уведомление во всех чатах
				for chatName := range m.chats {
					m.addMessageToChat(chatName, chatMessage{
						Sender:    "Система",
						Content:   "⚠️ " + protocolMsg.Content,
						Timestamp: time.Unix(protocolMsg.TimeStamp, 0),
						IsMe:      false,
					})
				}

				// Если нет активных чатов, показываем в текущем
				if m.activeChat == "" && len(m.contacts) > 0 {
					m.activeChat = m.contacts[0]
				}

				m.refreshViewport()
				return m, nil
			}

			// Другие системные сообщения
			targetChat := getTargetChat(protocolMsg, m.username)
			m.addMessageToChat(targetChat, chatMessage{
				Sender:    "Система",
				Content:   protocolMsg.Content,
				Timestamp: time.Unix(protocolMsg.TimeStamp, 0),
				IsMe:      false,
			})
			return m, nil
		}

		// Обычные сообщения
		targetChat := getTargetChat(protocolMsg, m.username)
		m.addMessageToChat(targetChat, chatMessage{
			Sender:    protocolMsg.Sender,
			Content:   protocolMsg.Content,
			Timestamp: time.Unix(protocolMsg.TimeStamp, 0),
			IsMe:      protocolMsg.Sender == m.username,
		})
		return m, nil
	}

	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)
	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	return m, tea.Batch(tiCmd, vpCmd)
}

func isSpecialKey(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyUp, tea.KeyDown, tea.KeyRight,
		tea.KeyEscape, tea.KeyTab, tea.KeyEnter:
		return true
	}
	return false
}

func (m *model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit

	case tea.KeyUp:
		m.navigateReply(-1)
		return m, nil

	case tea.KeyDown:
		m.navigateReply(1)
		return m, nil

	case tea.KeyRight:
		m.setForwardMessage()
		return m, nil

	case tea.KeyEscape:
		if m.replyTo != nil || m.forwardMsg != nil {
			m.replyTo = nil
			m.forwardMsg = nil
			m.refreshViewport()
		}
		return m, nil

	case tea.KeyTab:
		// Проверяем, вводит ли пользователь код эмодзи
		currentInput := m.textarea.Value()
		lastWord := getLastWord(currentInput)

		if strings.HasPrefix(lastWord, ":") {
			// Пытаемся автодополнить эмодзи
			completed := m.autocompleteEmoji(currentInput)
			if completed != currentInput {
				m.textarea.SetValue(completed)
				return m, nil
			}
		}
		// Если не эмодзи или не получилось дополнить - переключаем чат
		m.switchToNextContact()
		return m, nil

	case tea.KeyEnter:
		content := strings.TrimSpace(m.textarea.Value())
		m.textarea.Reset()

		if content == "" && m.forwardMsg == nil {
			return m, nil
		}

		// Проверяем, является ли ввод командой
		if m.handleCommand(content) {
			return m, nil
		}

		// Отправляем обычное сообщение
		m.sendMessage(content)
		return m, nil
	}

	return m, nil
}

func (m *model) View() string {
	mainLayout := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.renderSidebar(),
		m.viewport.View(),
	)
	emojiHints := m.renderEmojiHints()

	return fmt.Sprintf(
		"Вы вошли как: %s\n\n%s\n\n%s%s\n%s\n%s",
		m.username,
		mainLayout,
		m.renderStatusLine(),
		m.textarea.View(),
		emojiHints,
		"TAB - сменить чат | /add имя - начать чат | Ctrl+C - выход",
	)
}

// autocompleteEmoji пытается дополнить код эмодзи
func (m *model) autocompleteEmoji(input string) string {
	lastWord := getLastWord(input)
	if !strings.HasPrefix(lastWord, ":") {
		return input
	}

	suggestions := FindEmojiSuggestions(lastWord)
	if len(suggestions) == 1 {
		// Заменяем последнее слово на полный код эмодзи
		words := strings.Fields(input)
		words[len(words)-1] = suggestions[0]
		return strings.Join(words, " ") + " "
	}

	// Если несколько вариантов, показываем первый
	if len(suggestions) > 0 {
		words := strings.Fields(input)
		words[len(words)-1] = suggestions[0]
		return strings.Join(words, " ")
	}

	return input
}

func (m *model) renderEmojiHints() string {
	currentInput := m.textarea.Value()
	lastWord := getLastWord(currentInput)

	if !strings.HasPrefix(lastWord, ":") || len(lastWord) < 2 {
		return ""
	}

	suggestions := FindEmojiSuggestions(lastWord)
	if len(suggestions) == 0 {
		return ""
	}

	// Форматируем каждый вариант: эмодзи + код
	var items []string
	for _, code := range suggestions {
		emoji := EmojiMap[code]
		items = append(items, fmt.Sprintf("%s %s", emoji, code))
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true).
		Render(strings.Join(items, "  |  "))
}

// getLastWord возвращает последнее слово из текста
func getLastWord(text string) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}
	return words[len(words)-1]
}
