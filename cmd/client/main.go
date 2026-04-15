package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"messenger/internal/protocol"

	"google.golang.org/protobuf/proto"
)

func main() {
	// 1. Подключаемся к серверу
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatalf("Не удалось подключиться к серверу: %v", err)
	}
	defer conn.Close()

	// 2. Спрашиваем никнейм
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Введите ваш ник: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	// 3. Отправляем пакет авторизации (Тип 1)
	authReq := &protocol.AuthRequest{Username: username}
	payload, _ := proto.Marshal(authReq)
	if err := protocol.WritePacket(conn, 1, payload); err != nil {
		log.Fatalf("Ошибка отправки авторизации: %v", err)
	}

	fmt.Printf("Вы вошли как [%s]. Напишите что-нибудь...\n", username)

	// 4. ЗАПУСКАЕМ ФОНОВОЕ ЧТЕНИЕ
	// Эта горутина будет постоянно ждать сообщений от сервера
	go listenServer(conn)

	// 5. ОСНОВНОЙ ЦИКЛ ОТПРАВКИ
	// Программа будет "стоять" здесь и ждать твоего ввода
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ") // Приглашение к вводу
		if !scanner.Scan() {
			break
		}

		text := scanner.Text()
		if text == "" {
			continue
		}

		// Формируем пакет сообщения (Тип 2)
		chatMsg := &protocol.ChatMessage{
			SenderId: username,
			Text:     text,
		}

		msgPayload, _ := proto.Marshal(chatMsg)
		if err := protocol.WritePacket(conn, 2, msgPayload); err != nil {
			fmt.Println("Связь с сервером потеряна")
			break
		}
	}
}

// listenServer — функция, которая работает в фоне (в горутине)
func listenServer(conn net.Conn) {
	for {
		msgType, payload, err := protocol.ReadPacket(conn)
		if err != nil {
			fmt.Println("\nСоединение с сервером разорвано.")
			os.Exit(0) // Завершаем всё приложение, если сервер упал
		}

		// Если пришло обычное сообщение (Тип 2)
		if msgType == 2 {
			msg := &protocol.ChatMessage{}
			if err := proto.Unmarshal(payload, msg); err == nil {
				// Выводим сообщение от другого пользователя
				// \r и пробелы нужны, чтобы "затереть" текущую строку ввода "> "
				fmt.Printf("\r[%s]: %s\n> ", msg.SenderId, msg.Text)
			}
		}
	}
}
