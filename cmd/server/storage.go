package main

import (
	"database/sql"
	"fmt"
	"log"
	"messenger/protocol"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type Storage struct {
	db *sql.DB
}

func InitStorage(path string) *Storage {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatal(err)
	}

	// Создание таблиц
	queries := []string{
		`CREATE TABLE IF NOT EXISTS messages(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sender TEXT,
			recipient TEXT,
			content TEXT,
			timestamp INTEGER,
			action INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS system_logs(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_type TEXT,
			details TEXT,
			timestamp INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS users(
			username TEXT PRIMARY KEY,
			password_hash TEXT NOT NULL
		);`,
	}

	for _, query := range queries {
		if _, err = db.Exec(query); err != nil {
			log.Fatal(err)
		}
	}

	return &Storage{db: db}
}

// Системные логи
func (s *Storage) LogEvent(eventType, details string) {
	query := `INSERT INTO system_logs (event_type, details, timestamp) VALUES (?, ?, ?)`
	_, err := s.db.Exec(query, eventType, details, time.Now().Unix())
	if err != nil {
		fmt.Printf("Ошибка записи лога: %v\n", err)
	}
}

func (s *Storage) GetSystemLogs(limit int) ([]string, error) {
	query := `SELECT event_type, details, timestamp FROM system_logs 
	WHERE event_type NOT LIKE 'FILE_%' 
	ORDER BY timestamp DESC LIMIT ?`
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []string
	//итерация по результатам
	for rows.Next() {
		var eventType, details string
		var timestamp int64
		rows.Scan(&eventType, &details, &timestamp)

		t := time.Unix(timestamp, 0).Format("15:04:05")
		logEntry := fmt.Sprintf("[%s] %s: %s", t, eventType, details)
		logs = append(logs, logEntry)
	}
	return logs, nil
}

// Логи сообщений
func (s *Storage) GetMessageLogs(limit int) ([]string, error) {
	query := `SELECT sender, recipient, content, timestamp FROM messages ORDER BY timestamp DESC LIMIT ?`
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []string
	for rows.Next() {
		var sender, recipient, content string
		var timestamp int64
		rows.Scan(&sender, &recipient, &content, &timestamp)
		t := time.Unix(timestamp, 0).Format("15:04:05")
		logEntry := fmt.Sprintf("[%s] %s -> %s: %s", t, sender, recipient, content)
		logs = append(logs, logEntry)
	}
	return logs, err
}

// Сообщения
func (s *Storage) SaveMessage(msg protocol.Message) error {
	query := `INSERT INTO messages(sender, recipient, content, timestamp, action) VALUES (?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, msg.Sender, msg.Recipient, msg.Content, msg.TimeStamp, msg.Action)
	return err
}

func (s *Storage) GetHistory(username string) ([]protocol.Message, error) {
	query := `SELECT sender, recipient, content, timestamp, action
	 FROM messages WHERE recipient = ? OR sender = ?
	 OR (recipient LIKE '#%' AND recipient IN (SELECT DISTINCT recipient FROM messages WHERE sender = ?)) 
	 ORDER BY timestamp ASC`
	rows, err := s.db.Query(query, username, username, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []protocol.Message
	for rows.Next() {
		var m protocol.Message
		err := rows.Scan(&m.Sender, &m.Recipient, &m.Content, &m.TimeStamp, &m.Action)
		if err != nil {
			return nil, err
		}
		history = append(history, m)
	}
	return history, nil
}

// Пользователи
func (s *Storage) RegisterUser(username, password string) error {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	_, err := s.db.Exec("INSERT INTO users(username, password_hash) VALUES (?, ?)", username, string(hash))
	return err
}

func (s *Storage) CheckPassword(username, password string) bool {
	var hash string
	err := s.db.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&hash)
	if err != nil {
		return false
	}
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (s *Storage) UserExists(username string) bool {
	var name string
	err := s.db.QueryRow("SELECT username FROM users WHERE username = ?", username).Scan(&name)
	return err == nil
}

// LogFileEvent логирует событие передачи файла
func (s *Storage) LogFileEvent(eventType, sender, recipient, fileName string, fileSize int64, fileID string) {
	query := `INSERT INTO system_logs (event_type, details, timestamp) VALUES (?, ?, ?)`
	details := fmt.Sprintf("File: %s | From: %s -> To: %s | Size: %d bytes | ID: %s",
		fileName, sender, recipient, fileSize, fileID)
	_, err := s.db.Exec(query, eventType, details, time.Now().Unix())
	if err != nil {
		fmt.Printf("Ошибка записи лога файла: %v\n", err)
	}
}

// GetFileLogs возвращает логи файловых операций
func (s *Storage) GetFileLogs(limit int) ([]string, error) {
	query := `SELECT event_type, details, timestamp FROM system_logs 
	          WHERE event_type LIKE 'FILE_%' 
	          ORDER BY timestamp DESC LIMIT ?`
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []string
	for rows.Next() {
		var eventType, details string
		var timestamp int64
		rows.Scan(&eventType, &details, &timestamp)
		t := time.Unix(timestamp, 0).Format("15:04:05")
		logEntry := fmt.Sprintf("[%s] %s: %s", t, eventType, details)
		logs = append(logs, logEntry)
	}
	return logs, nil
}
