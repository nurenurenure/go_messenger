package main

import (
	"fmt"
	"log"
	"net"

	// Убедись, что пути соответствуют твоему go.mod
	"messenger/internal/protocol"
	"messenger/internal/server"

	"google.golang.org/protobuf/proto"
)

func main() {
	// 1. Создаем ОДИН общий Хаб для всех клиентов
	myHub := server.NewHub()

	// 2. Открываем порт для прослушивания
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
	defer listener.Close()

	fmt.Println("Сервер мессенджера запущен на :8080")

	// 3. Основной цикл: принимаем новых людей бесконечно
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Ошибка подключения: %v", err)
			continue
		}

		// 4. МНОГОПОТОЧНОСТЬ: для каждого клиента своя горутина
		// Мы передаем в неё и соединение, и ссылку на наш Хаб
		go handleConnection(conn, myHub)
	}
}

// handleConnection — это «индивидуальный менеджер» для каждого клиента
func handleConnection(conn net.Conn, hub *server.Hub) {
	defer conn.Close()

	var currentUsername string

	// Гарантируем, что если клиент уйдет, он удалится из списка онлайн
	defer func() {
		if currentUsername != "" {
			hub.Unregister(currentUsername)
			fmt.Printf("%s покинул чат\n", currentUsername)
		}
	}()

	for {
		// Читаем пакет по нашему бинарному протоколу (8 байт заголовок + тело)
		msgType, payload, err := protocol.ReadPacket(conn)
		if err != nil {
			// Если произошла ошибка (например, клиент просто закрыл окно), выходим из цикла
			return
		}

		switch msgType {
		case 1: // ТИП 1: АВТОРИЗАЦИЯ
			req := &protocol.AuthRequest{}
			if err := proto.Unmarshal(payload, req); err != nil {
				log.Printf("Ошибка распаковки AuthRequest: %v", err)
				continue
			}

			currentUsername = req.Username
			// Регистрируем имя и соединение в Хабе
			hub.Register(currentUsername, conn)
			fmt.Printf("%s вошел в систему\n", currentUsername)

		case 2: // ТИП 2: ЧАТ-СООБЩЕНИЕ
			// Серверу даже не обязательно распаковывать (Unmarshal) сообщение,
			// если он просто хочет переслать его другим "как есть".
			// Мы просто вызываем Broadcast, и Хаб рассылает эти байты всем остальным.
			hub.Broadcast(msgType, payload)

			// Но для логов на сервере можем и распаковать:
			msg := &protocol.ChatMessage{}
			proto.Unmarshal(payload, msg)
			fmt.Printf("Сообщение от %s: %s\n", msg.SenderId, msg.Text)
		}
	}
}
