package main

import (
	"database/sql"
	"fmt"
	"log"
	"messenger/protocol"
	"time"
)

type Storage struct {
	db *sql.DB
}

// открыть файл базы и создать таблицу, если ее нет
func InitStorage(path string) *Storage {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatal(err)
	}

	//создать таблицу для сообщений
	query := `
	CREATE TABLE IF NOT EXISTS messages(
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	sender TEXT,
	recipient TEXT,
	content TEXT,
	timestamp INTEGER,
	action TEXT
	);`

	//таблица для системных логов

	queryLogs := `
	CREATE TABLE IF NOT EXISTS system_logs(
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	event_type TEXT,
	details TEXT,
	timestamp INTEGER
	);`

	_, err = db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(queryLogs)
	if err != nil {
		log.Fatal(err)
	}
	return &Storage{db: db}

}

// записать системный лог
func (s *Storage) LogEvent(eventType, details string) {
	query := `INSERT INTO system_logs (event_type, details, timestamp) VALUES (?, ?, ?)`
	_, err := s.db.Exec(query, eventType, details, time.Now().Unix())
	if err != nil {
		fmt.Printf("Ошибка записи лога: %v\n", err)
	}
}

// вернуть ограниченное количество логов для админа
func (s *Storage) GetSystemLogs(limit int) ([]string, error) {
	query := `SELECT event_type, details, timestamp FROM system_logs ORDER BY timestamp DESC LIMIT ?`
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

		//форматированная строка лога
		t := time.Unix(timestamp, 0).Format("15:04:05")
		logEntry := fmt.Sprintf("[%s] %s: %s", t, eventType, details)
		logs = append(logs, logEntry)
	}
	return logs, nil
}

// вернуть ограниченное количество последних сообщений для админа
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

// сохранить сообщение в базу
func (s *Storage) SaveMessage(msg protocol.Message) error {
	query := `INSERT INTO messages(sender, recipient, content, timestamp, action) VALUES (?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, msg.Sender, msg.Recipient, msg.Content, msg.TimeStamp, msg.Action)
	return err
}

// достать историю сообщений для конкретного пользователя
func (s *Storage) GetHistory(username string) ([]protocol.Message, error) {
	query := `SELECT sender, recipient, content, timestamp, action FROM messages WHERE recipient = ? OR sender = ? ORDER BY timestamp ASC`
	rows, err := s.db.Query(query, username, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []protocol.Message
	for rows.Next() {
		//гарантия чистого объекта
		var m protocol.Message
		err := rows.Scan(&m.Sender, &m.Recipient, &m.Content, &m.TimeStamp, &m.Action)
		if err != nil {
			return nil, err
		}
		history = append(history, m)
	}
	return history, nil
}
