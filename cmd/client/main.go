package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"messenger/protocol"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	conf := &tls.Config{InsecureSkipVerify: true}
	conn, err := tls.Dial("tcp", "localhost:8080", conf)
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

	// Инициализация БД
	db, err := initDB(username, password)
	if err != nil {
		log.Fatal("Не удалось открыть базу данных", err)
	}
	defer db.Close()

	// Создаем модель
	m := InitialModel(conn, username, db)

	// Создаем программу
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Запускаем прослушивание сообщений
	go func() {
		for {
			msgType, payload, err := protocol.ReadPacket(conn)
			if err != nil {
				break
			}

			// Обработка файловых сообщений
			switch msgType {
			case protocol.TypeFileRequest,
				protocol.TypeFileHeader,
				protocol.TypeFileChunk,
				protocol.TypeFileComplete,
				protocol.TypeFileError:
				m.fileReceiver.HandleFileMessage(msgType, payload)
				continue
			}

			// Обычные сообщения
			if msgType == protocol.TypeChat {
				var incoming protocol.Message
				if json.Unmarshal(payload, &incoming) == nil {
					p.Send(serverMsg(incoming))
				}
			}
		}
	}()

	// Запускаем приложение
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
