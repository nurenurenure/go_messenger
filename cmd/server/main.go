package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"messenger/internal/protocol"

	"google.golang.org/protobuf/proto"
)

func main() {
	// 1. Запуск слушателя на порту 8080
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Не удалось запустить сервер: %v", err)
	}

	// Закрываем слушатель при выходе из main
	defer listener.Close()

	fmt.Println("Сервер мессенджера запущен на :8080")
	fmt.Println("Ожидание подключений...")

	// Канал для отслеживания системных прерываний
	// Это нужно для Graceful Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Основной цикл приема соединений
	go func() {
		for {
			// 2. Принимаем новое входящее соединение
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Ошибка при приеме соединения: %v", err)
				continue
			}

			// 3. МНОГОПОТОЧНОСТЬ: запускаем отдельную горутину для каждого клиента
			go handleConnection(conn)
		}
	}()

	// Ждем сигнала завершения, чтобы сервер не закрылся сразу
	<-sigChan
	fmt.Println("\nЗавершение работы сервера...")
}

// handleConnection управляет жизненным циклом одного клиента
func handleConnection(conn net.Conn) {
	// Закрываем соединение, когда функция завершится
	defer conn.Close()

	clientAddr := conn.RemoteAddr().String()
	fmt.Printf("Новое соединение: %s\n", clientAddr)

	for {
		// 4. Читаем пакет
		msgType, payload, err := protocol.ReadPacket(conn)
		if err != nil {
			fmt.Printf("Клиент %s отключился или произошла ошибка: %v\n", clientAddr, err)
			return
		}

		// 5. Диспетчеризация сообщений
		switch msgType {
		case 1: // Тип 1: Запрос на авторизацию
			handleAuth(payload)
		case 2: // Тип 2: Текстовое сообщение
			handleChatMessage(payload)
		default:
			fmt.Printf("Получен неизвестный тип сообщения: %d\n", msgType)
		}
	}
}

// Функции-обработчики конкретных типов данных

func handleAuth(payload []byte) {
	// Создаем пустую структуру из сгенерированного файла
	req := &protocol.AuthRequest{}

	// Распаковываем байты Protobuf в структуру Go
	if err := proto.Unmarshal(payload, req); err != nil {
		log.Printf("Ошибка десериализации AuthRequest: %v", err)
		return
	}

	fmt.Printf("Попытка входа: Пользователь='%s'\n", req.Username)
}

func handleChatMessage(payload []byte) {
	msg := &protocol.ChatMessage{}

	if err := proto.Unmarshal(payload, msg); err != nil {
		log.Printf("Ошибка десериализации ChatMessage: %v", err)
		return
	}

	fmt.Printf("Сообщение от %s: %s\n", msg.SenderId, msg.Text)
}
