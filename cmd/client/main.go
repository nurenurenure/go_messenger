package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"messenger/protocol"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Не удалось подключиться к серверу:", err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Введите Ваш никнейм: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	authMsg := protocol.Message{Sender: username}

	protocol.WritePacket(conn, 1, authMsg)
	fmt.Printf("Вы вошли как %s\n", username)

	go func() {
		for {
			msgType, payload, err := protocol.ReadPacket(conn)
			if err != nil {
				fmt.Println("\nСвязь с сервером потеряна:")
				os.Exit(1)
			}

			if msgType == 2 {
				var incoming protocol.Message
				json.Unmarshal(payload, &incoming)
				fmt.Printf("\n[%s]: %s\n> ", incoming.Sender, incoming.Content)
			}

		}
	}()

	fmt.Println("Для отправки введите: получатель:сообщение")
	for {
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		parts := strings.SplitN(input, ":", 2)

		recipient := strings.TrimSpace(parts[0])
		content := strings.TrimSpace(parts[1])

		msg := protocol.Message{
			Sender:    username,
			Recipient: recipient,
			Content:   content,
			TimeStamp: time.Now().Unix(),
			Action:    protocol.ActionSendMessage,
		}

		err := protocol.WritePacket(conn, 2, msg)
		if err != nil {
			fmt.Println("Ошибка при отправке пакета:", err)
			break
		}
	}

}
