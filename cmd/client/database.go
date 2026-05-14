package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger/v4"
)

func initDB(username string) (*badger.DB, error) {
	path := filepath.Join("user_data", "db_"+username)

	if err := os.MkdirAll("user_data", 0755); err != nil {
		return nil, err
	}

	opts := badger.DefaultOptions(path)

	//ключ шифрования (пока что заглушка)
	opts.EncryptionKey = []byte("a-very-secret-key-32-chars=long!")
	opts.IndexCacheSize = 10 << 20 //10MB

	return badger.Open(opts)
}

func saveToDB(db *badger.DB, chatName string, msg chatMessage) {
	db.Update(func(txn *badger.Txn) error {
		//ключ: char_username:timestamp (чтобы обеспечить сортировку по времени)
		key := []byte(fmt.Sprintf("%s:%d", chatName, msg.Timestamp.UnixNano()))

		payload, _ := json.Marshal(msg)

		return txn.Set(key, payload)

	})
}

func loadHistory(db *badger.DB, chatName string) []chatMessage {
	var history []chatMessage
	db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := []byte(chatName + ":")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			item.Value(func(v []byte) error {
				var msg chatMessage
				json.Unmarshal(v, &msg)
				history = append(history, msg)
				return nil
			})
		}
		return nil
	})
	return history
}
