package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dgraph-io/badger/v4"
)

func initDB(username, rawPassword string) (*badger.DB, error) {
	path := filepath.Join("user_data", "db_"+username)

	if err := os.MkdirAll("user_data", 0755); err != nil {
		return nil, err
	}
	//Пароль как ключ шифрования
	keyHash := sha256.Sum256([]byte(rawPassword))

	opts := badger.DefaultOptions(path)

	//ключ шифрования
	opts.EncryptionKey = keyHash[:]
	opts.IndexCacheSize = 100 << 20

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

// deleteChatHistory удаляет все сообщения конкретного чата из локальной БД
func deleteChatHistory(db *badger.DB, chatName string) error {
	db.Update(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := []byte(chatName + ":")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			err := txn.Delete(it.Item().KeyCopy(nil))
			if err != nil {
				return err
			}
		}
		return nil
	})
	return nil
}

// сканирует все ключи в БД.
// пропускает системные ключи (system:last_sync).
// парсит имя чата как часть до первого :.
// для каждого уникального имени чата: добавляет в список контактов и загружает всю историю через loadHistory.
func loadAllContacts(db *badger.DB) ([]string, map[string][]chatMessage) {
	contacts := []string{}
	chats := make(map[string][]chatMessage)
	seen := make(map[string]bool)

	db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			key := string(it.Item().Key())
			//игнор системных ключей
			if strings.HasPrefix(key, "system:") {
				continue
			}
			parts := strings.Split(key, ":")
			if len(parts) > 0 && !seen[parts[0]] {
				name := parts[0]
				seen[name] = true
				contacts = append(contacts, name)
				chats[name] = loadHistory(db, name)
			}
		}
		return nil
	})

	return contacts, chats
}

// Сохраняем время последнего сообщения
func updateLastSync(db *badger.DB, timestamp int64) {
	db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("system:last_sync"), []byte(fmt.Sprintf("%d", timestamp)))
	})
}

// получаем время последнего сообщения
func getLastSync(db *badger.DB) int64 {
	var lastSync int64 = 0
	db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("system:last_sync"))
		if err == nil {
			item.Value(func(v []byte) error {
				fmt.Sscanf(string(v), "%d", &lastSync)
				return nil
			})
		}
		return err
	})
	return lastSync

}
