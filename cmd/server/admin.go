package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
)

// handleAdminCommands обрабатывает команды администратора
func handleAdminCommands(srv *Server, cancel context.CancelFunc) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Доступные команды: /logs, /msglogs, /filelogs, /users, /exit")

	for scanner.Scan() {
		command := scanner.Text()
		srv.executeAdminCommand(command, cancel)
	}
}

// executeAdminCommand выполняет конкретную админскую команду
func (s *Server) executeAdminCommand(command string, cancel context.CancelFunc) {
	switch command {
	case "/logs":
		s.showSystemLogs()
	case "/users":
		s.showOnlineUsers()
	case "/msglogs":
		s.showMessageLogs()
	case "/filelogs":
		s.showFileLogs()
	case "/exit":
		fmt.Println("Завершение работы сервера...")
		s.storage.LogEvent("SERVER_STOP", "Сервер остановлен администратором")
		cancel()
	default:
		fmt.Println("Неизвестная команда")
	}
}

// showSystemLogs показывает системные логи
func (s *Server) showSystemLogs() {
	logs, err := s.storage.GetSystemLogs(20)
	if err != nil {
		fmt.Println("Ошибка получения логов:", err)
		return
	}
	for _, l := range logs {
		fmt.Println(l)
	}
}

// showOnlineUsers показывает пользователей онлайн
func (s *Server) showOnlineUsers() {
	fmt.Println("В сети:", s.getOnlineCount())
	for _, name := range s.getOnlineUsers() {
		fmt.Println("-", name)
	}
}

// showMessageLogs показывает историю сообщений
func (s *Server) showMessageLogs() {
	msgLogs, err := s.storage.GetMessageLogs(50)
	if err != nil {
		fmt.Println("Ошибка получения истории сообщений:", err)
		return
	}

	if len(msgLogs) == 0 {
		fmt.Println("История сообщений пуста")
		return
	}

	for _, l := range msgLogs {
		fmt.Println(l)
	}
}
func (s *Server) showFileLogs() {
	logs, err := s.storage.GetFileLogs(50)
	if err != nil {
		fmt.Println("Ошибка получения логов файлов:", err)
		return
	}

	if len(logs) == 0 {
		fmt.Println("История файловых операций пуста")
		return
	}

	fmt.Println("=== Логи файловых операций ===")
	for _, l := range logs {
		fmt.Println(l)
	}
}
