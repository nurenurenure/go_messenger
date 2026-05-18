package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"messenger/certgen"
)

func main() {
	// Инициализация сервера и БД
	srv := NewServer()
	srv.storage = InitStorage("./messenger.db")
	defer func() {
		srv.storage.LogEvent("SERVER_STOP", "Сервер остановлен")
		srv.storage.db.Close()
	}()

	// Проверка и генерация TLS сертификатов
	certFile := "server.crt"
	keyFile := "server.key"
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		fmt.Println("TLS сертификаты не найдены. Генерация...")
		if err := certgen.GenerateCerts(); err != nil {
			log.Fatal("Критическая ошибка при генерации сертификатов.", err)
		}
		fmt.Println("Сертификаты успешно созданы")
	}

	// Загрузка сертификатов
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal("Ошибка загрузки TLS ключей", err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	// Создаем контекст с отменой для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запуск админской консоли
	go handleAdminCommands(srv, cancel)

	// Запуск сервера
	listener, err := tls.Listen("tcp", ":8080", tlsConfig)
	if err != nil {
		log.Fatal("Ошибка запуска TLS слушателя", err)
	}

	srv.storage.LogEvent("SERVER_START", "Сервер запущен на :8080")
	fmt.Println("Сервер запущен на :8080")
	fmt.Println("Для остановки нажмите Ctrl+C или введите /exit")

	// Канал для graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// WaitGroup для отслеживания активных соединений
	var wg sync.WaitGroup

	// Запускаем принятие соединений в горутине
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					// Сервер завершает работу, выходим из цикла
					return
				default:
					// Проверяем, не закрыт ли слушатель
					if strings.Contains(err.Error(), "use of closed network connection") {
						return
					}
					fmt.Println("Ошибка подключения:", err)
					continue
				}
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				srv.handleClient(conn)
			}()
		}
	}()

	// Ожидаем сигнал завершения или админскую команду /exit
	select {
	case sig := <-shutdown:
		fmt.Printf("\nПолучен сигнал: %v\n", sig)
	case <-ctx.Done():
		fmt.Println("\nПолучена команда завершения")
	}

	// Начинаем graceful shutdown
	fmt.Println("Завершение работы сервера...")

	// Уведомляем клиентов
	srv.notifyClientsAboutShutdown()

	time.Sleep(500 * time.Millisecond)

	// Закрываем слушатель, чтобы прекратить прием новых соединений
	listener.Close()

	// Ждем завершения текущих соединений с таймаутом
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("Все соединения закрыты")
	case <-time.After(5 * time.Second):
		fmt.Println("Таймаут ожидания закрытия соединений")
	}

	// Закрываем все оставшиеся соединения
	srv.closeAllConnections()

	fmt.Println("Сервер остановлен")
}
