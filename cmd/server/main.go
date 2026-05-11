package main

import (
	"fmt"
	"messenger/protocol"
	"net"
)

func main() {
	listener, _ := net.Listen("tcp", ":8080")
	fmt.Println("Сервер запущен на :8080")

	for {
		conn, _ := listener.Accept()
		// горутина для каждого клиента
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()
	for {
		msgType, payload, err := protocol.ReadPacket(conn)
		if err != nil {
			fmt.Println("Клиент отключился")
			return
		}

		fmt.Printf("Получен пакет типа %d: %s/n", msgType, string(payload))
	}
}
