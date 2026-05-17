package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"

	"messenger/certgen"
)

func main() {
	// Инициализация сервера и БД
	srv := NewServer()
	srv.storage = InitStorage("./messenger.db")
	defer srv.storage.LogEvent("SERVER_STOP", "Сервер остановлен пользователем.")

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

	// Запуск админской консоли
	go handleAdminCommands(srv)

	// Запуск сервера
	srv.storage.LogEvent("SERVER_START", "Сервер запущен на :8080")
	fmt.Println("Сервер запущен...")

	listener, err := tls.Listen("tcp", ":8080", tlsConfig)
	if err != nil {
		log.Fatal("Ошибка запуска TLS слушателя", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Ошибка подключения:", err)
			continue
		}
		go srv.handleClient(conn)
	}
}
