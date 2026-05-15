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

	// Запуск приложения
	p := tea.NewProgram(InitialModel(conn, username, db), tea.WithAltScreen())

	// Слушаем сообщения от сервера в отдельной горутине
	go func() {
		for {
			msgType, payload, err := protocol.ReadPacket(conn)
			if err != nil {
				break
			}
			if msgType == protocol.TypeChat {
				var incoming protocol.Message
				if json.Unmarshal(payload, &incoming) == nil {
					p.Send(serverMsg(incoming))
				}
			}
		}
	}()

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
