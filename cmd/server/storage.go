package main

import (
	"database/sql"
	"log"
	"messenger/protocol"
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

	_, err = db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
	return &Storage{db: db}

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
