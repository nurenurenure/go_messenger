package main

import (
	"fmt"
	"messenger/protocol"
	"net"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Не удалось подключиться к серверу:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Подключено к серверу!")

	testMsg := map[string]string{
		"sender":  "Anna",
		"content": "你好，我的朋友们",
	}

	err = protocol.WritePacket(conn, 2, testMsg)
	if err != nil {
		fmt.Println("Ошибка отправки:", err)
	}

}
