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
	event_typr TEXT, -- "SERVER_START", "USER_LOGIN", "USER_LOGOUT", "SERVER_STOP"
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
func (s *Storage) LogEvent(evenType, details string) {
	query := `INSERT INTO system_logs (event_type, details, time_stamp) VALUES (?, ?, ?)`
	_, err := s.db.Exec(query, evenType, details, time.Now().Unix())
	if err != nil {
		fmt.Printf("Ошибка записи лога: %v\n", err)
	}
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
