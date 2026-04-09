package main

import (
	"fmt"
	"log"
	"net"
	"time"

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

	fmt.Println("Подключено к серверу!")

	// 2. Формируем данные для авторизации (Protobuf)
	authReq := &protocol.AuthRequest{
		Username: "Anna_Developer",
		Password: "super_secret_password",
	}

	// 3. Маршалим (сериализуем) структуру в байты
	payload, err := proto.Marshal(authReq)
	if err != nil {
		log.Fatalf("Ошибка сериализации: %v", err)
	}

	// 4. Отправляем пакет через наш кастомный протокол
	// Тип 1 соответствует handleAuth на сервере
	err = protocol.WritePacket(conn, 1, payload)
	if err != nil {
		log.Fatalf("Ошибка отправки пакета: %v", err)
	}

	fmt.Println("Запрос на авторизацию отправлен!")

	// 5. Небольшая пауза, чтобы сервер успел обработать,
	// прежде чем мы закроем соединение и выйдем
	time.Sleep(1 * time.Second)
}
