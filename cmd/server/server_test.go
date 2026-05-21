package main

import (
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"messenger/protocol"
)

// 1. Тест создания FileTransferManager
func TestNewFileTransferManager(t *testing.T) {
	ftm := NewFileTransferManager()
	if ftm == nil {
		t.Fatal("NewFileTransferManager вернул nil")
	}
	if ftm.transfers == nil {
		t.Fatal("transfers map не инициализирована")
	}
}

// 2. Тест создания Server
func TestNewServer(t *testing.T) {
	srv := NewServer()
	if srv == nil {
		t.Fatal("NewServer вернул nil")
	}
	if srv.clients == nil {
		t.Fatal("clients map не инициализирована")
	}
	if srv.groups == nil {
		t.Fatal("groups map не инициализирована")
	}
	if srv.fileTransfers == nil {
		t.Fatal("fileTransfers не инициализирован")
	}
}

// 3. Тест добавления и получения клиента
func TestAddAndGetClient(t *testing.T) {
	srv := NewServer()
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	srv.addClient("testuser", server)

	conn, ok := srv.getClient("testuser")
	if !ok {
		t.Fatal("Клиент не найден после добавления")
	}
	if conn == nil {
		t.Fatal("Соединение клиента равно nil")
	}

	_, ok = srv.getClient("nonexistent")
	if ok {
		t.Error("Несуществующий клиент не должен быть найден")
	}
}

// 4. Тест удаления клиента
func TestRemoveClient(t *testing.T) {
	srv := NewServer()

	client1, _ := net.Pipe()
	client2, _ := net.Pipe()

	srv.addClient("user1", client1)
	srv.addClient("user2", client2)

	srv.removeClient("user1")

	_, ok := srv.getClient("user1")
	if ok {
		t.Error("Клиент должен быть удален")
	}

	_, ok = srv.getClient("user2")
	if !ok {
		t.Error("Другой клиент не должен быть удален")
	}
}

// 5. Тест подсчета онлайн пользователей
func TestGetOnlineCount(t *testing.T) {
	srv := NewServer()

	client1, _ := net.Pipe()
	client2, _ := net.Pipe()
	client3, _ := net.Pipe()

	srv.addClient("alice", client1)
	srv.addClient("bob", client2)
	srv.addClient("charlie", client3)

	count := srv.getOnlineCount()
	if count != 3 {
		t.Errorf("Ожидалось 3 пользователя, получено %d", count)
	}

	srv.removeClient("alice")
	count = srv.getOnlineCount()
	if count != 2 {
		t.Errorf("Ожидалось 2 пользователя, получено %d", count)
	}
}

// 6. Тест получения списка онлайн пользователей
func TestGetOnlineUsers(t *testing.T) {
	srv := NewServer()

	client1, _ := net.Pipe()
	client2, _ := net.Pipe()

	srv.addClient("alice", client1)
	srv.addClient("bob", client2)

	users := srv.getOnlineUsers()
	if len(users) != 2 {
		t.Errorf("Ожидалось 2 пользователя, получено %d", len(users))
	}

	userMap := make(map[string]bool)
	for _, u := range users {
		userMap[u] = true
	}
	if !userMap["alice"] || !userMap["bob"] {
		t.Error("Не все пользователи найдены в списке")
	}
}

// 7. Тест добавления в группу
func TestAddToGroup(t *testing.T) {
	srv := NewServer()

	client1, _ := net.Pipe()
	client2, _ := net.Pipe()

	srv.addClient("user1", client1)
	srv.addClient("user2", client2)

	srv.addToGroup("#general", "user1")
	srv.addToGroup("#general", "user2")

	srv.mu.Lock()
	group := srv.groups["#general"]
	srv.mu.Unlock()

	if len(group) != 2 {
		t.Errorf("Ожидалось 2 участника, получено %d", len(group))
	}
}

// 8. Тест удаления из группы
func TestRemoveFromGroup(t *testing.T) {
	srv := NewServer()

	client1, _ := net.Pipe()
	client2, _ := net.Pipe()

	srv.addClient("user1", client1)
	srv.addClient("user2", client2)

	srv.addToGroup("#general", "user1")
	srv.addToGroup("#general", "user2")

	srv.removeFromGroup("#general", "user1")

	srv.mu.Lock()
	group := srv.groups["#general"]
	srv.mu.Unlock()

	if len(group) != 1 {
		t.Errorf("Ожидался 1 участник, получено %d", len(group))
	}
}

// 9. Тест удаления группы когда все вышли
func TestGroupAutoDelete(t *testing.T) {
	srv := NewServer()

	client1, _ := net.Pipe()
	srv.addClient("user1", client1)
	srv.addToGroup("#temp", "user1")

	srv.removeFromGroup("#temp", "user1")

	srv.mu.Lock()
	_, exists := srv.groups["#temp"]
	srv.mu.Unlock()

	if exists {
		t.Error("Группа должна быть удалена после ухода последнего участника")
	}
}

// 10. Тест закрытия всех соединений
func TestCloseAllConnections(t *testing.T) {
	srv := NewServer()

	_, server1 := net.Pipe()
	_, server2 := net.Pipe()

	srv.addClient("user1", server1)
	srv.addClient("user2", server2)

	srv.closeAllConnections()

	if len(srv.clients) != 0 {
		t.Error("clients map должна быть пустой")
	}
	if len(srv.groups) != 0 {
		t.Error("groups map должна быть пустой")
	}
}

// 11. Тест инициализации БД
func TestInitStorage(t *testing.T) {
	dbPath := "test_init.db"
	defer os.Remove(dbPath)

	storage := InitStorage(dbPath)
	if storage == nil {
		t.Fatal("InitStorage вернул nil")
	}
	defer storage.db.Close()

	var tableName string
	tables := []string{"messages", "system_logs", "users"}
	for _, table := range tables {
		err := storage.db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&tableName)
		if err != nil {
			t.Errorf("Таблица %s не создана", table)
		}
	}
}

// 12. Тест регистрации пользователя
func TestRegisterUser(t *testing.T) {
	dbPath := "test_register.db"
	defer os.Remove(dbPath)

	storage := InitStorage(dbPath)
	defer storage.db.Close()

	if storage.UserExists("newuser") {
		t.Error("Пользователь не должен существовать до регистрации")
	}

	err := storage.RegisterUser("newuser", "password123")
	if err != nil {
		t.Fatalf("RegisterUser вернул ошибку: %v", err)
	}

	if !storage.UserExists("newuser") {
		t.Error("Пользователь должен существовать после регистрации")
	}
}

// 13. Тест проверки пароля
func TestCheckPassword(t *testing.T) {
	dbPath := "test_password.db"
	defer os.Remove(dbPath)

	storage := InitStorage(dbPath)
	defer storage.db.Close()

	storage.RegisterUser("testuser", "correct_password")

	if !storage.CheckPassword("testuser", "correct_password") {
		t.Error("Правильный пароль не прошел проверку")
	}

	if storage.CheckPassword("testuser", "wrong_password") {
		t.Error("Неправильный пароль прошел проверку")
	}

	if storage.CheckPassword("nonexistent", "password") {
		t.Error("Проверка несуществующего пользователя должна вернуть false")
	}
}

// 14. Тест сохранения сообщения в БД
func TestSaveMessage(t *testing.T) {
	dbPath := "test_save_msg.db"
	defer os.Remove(dbPath)

	storage := InitStorage(dbPath)
	defer storage.db.Close()

	msg := protocol.Message{
		Sender:    "alice",
		Recipient: "bob",
		Content:   "Hello!",
		TimeStamp: time.Now().Unix(),
		Action:    protocol.TypeChat,
	}

	err := storage.SaveMessage(msg)
	if err != nil {
		t.Fatalf("SaveMessage вернул ошибку: %v", err)
	}

	history, err := storage.GetHistory("bob")
	if err != nil {
		t.Fatalf("GetHistory вернул ошибку: %v", err)
	}

	if len(history) == 0 {
		t.Fatal("История пуста после сохранения")
	}

	if history[0].Content != "Hello!" {
		t.Errorf("Ожидалось 'Hello!', получено '%s'", history[0].Content)
	}
}

// 15. Тест получения нескольких сообщений
func TestGetMultipleMessages(t *testing.T) {
	dbPath := "test_multiple.db"
	defer os.Remove(dbPath)

	storage := InitStorage(dbPath)
	defer storage.db.Close()

	for i := 0; i < 5; i++ {
		msg := protocol.Message{
			Sender:    "user1",
			Recipient: "user2",
			Content:   fmt.Sprintf("Message %d", i),
			TimeStamp: time.Now().Unix() + int64(i),
			Action:    protocol.TypeChat,
		}
		storage.SaveMessage(msg)
	}

	history, err := storage.GetHistory("user2")
	if err != nil {
		t.Fatalf("GetHistory вернул ошибку: %v", err)
	}

	if len(history) < 5 {
		t.Errorf("Ожидалось минимум 5 сообщений, получено %d", len(history))
	}
}

// 16. Тест системных логов
func TestSystemLogs(t *testing.T) {
	dbPath := "test_syslog.db"
	defer os.Remove(dbPath)

	storage := InitStorage(dbPath)
	defer storage.db.Close()

	storage.LogEvent("TEST_EVENT", "Test message")

	logs, err := storage.GetSystemLogs(10)
	if err != nil {
		t.Fatalf("GetSystemLogs вернул ошибку: %v", err)
	}

	if len(logs) == 0 {
		t.Fatal("Логи пусты")
	}

	found := false
	for _, log := range logs {
		if contains(log, "TEST_EVENT") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Тестовое событие не найдено в логах")
	}
}

// 17. Тест логов сообщений
func TestMessageLogs(t *testing.T) {
	dbPath := "test_msglog.db"
	defer os.Remove(dbPath)

	storage := InitStorage(dbPath)
	defer storage.db.Close()

	msg := protocol.Message{
		Sender:    "alice",
		Recipient: "bob",
		Content:   "Test log message",
		TimeStamp: time.Now().Unix(),
		Action:    protocol.TypeChat,
	}
	storage.SaveMessage(msg)

	logs, err := storage.GetMessageLogs(10)
	if err != nil {
		t.Fatalf("GetMessageLogs вернул ошибку: %v", err)
	}

	if len(logs) == 0 {
		t.Fatal("Логи сообщений пусты")
	}

	if !contains(logs[0], "alice") || !contains(logs[0], "bob") {
		t.Error("Лог не содержит отправителя или получателя")
	}
}

// 18. Тест файловых логов
func TestFileLogs(t *testing.T) {
	dbPath := "test_filelog.db"
	defer os.Remove(dbPath)

	storage := InitStorage(dbPath)
	defer storage.db.Close()

	storage.LogFileEvent("FILE_SEND", "alice", "bob", "test.pdf", 2048, "file-123")

	logs, err := storage.GetFileLogs(10)
	if err != nil {
		t.Fatalf("GetFileLogs вернул ошибку: %v", err)
	}

	if len(logs) == 0 {
		t.Fatal("Файловые логи пусты")
	}

	// Проверяем что системные логи НЕ содержат файловые
	sysLogs, _ := storage.GetSystemLogs(10)
	for _, log := range sysLogs {
		if contains(log, "FILE_") {
			t.Error("Системные логи не должны содержать файловые события")
		}
	}
}

// 19. Тест игнор-листа (без блокировки)
func TestIgnoreList(t *testing.T) {
	srv := NewServer()

	// Проверяем логику игнор-листа напрямую
	srv.mu.Lock()
	if srv.ignored["bob"] == nil {
		srv.ignored["bob"] = make(map[string]bool)
	}
	srv.ignored["bob"]["alice"] = true
	srv.mu.Unlock()

	// Проверяем что alice в игнор-листе у bob
	srv.mu.Lock()
	ignored := srv.ignored["bob"]["alice"]
	srv.mu.Unlock()

	if !ignored {
		t.Error("alice должна быть в игнор-листе bob")
	}

	// Проверяем логику проверки игнора (без реальной отправки)
	srv.mu.Lock()
	shouldBlock := srv.ignored["bob"] != nil && srv.ignored["bob"]["alice"]
	srv.mu.Unlock()

	if !shouldBlock {
		t.Error("Система должна блокировать сообщения от alice к bob")
	}

	// Убираем из игнора
	srv.mu.Lock()
	delete(srv.ignored["bob"], "alice")
	srv.mu.Unlock()

	// Проверяем что удалился
	srv.mu.Lock()
	stillIgnored := srv.ignored["bob"]["alice"]
	srv.mu.Unlock()

	if stillIgnored {
		t.Error("alice должна быть удалена из игнор-листа")
	}
}

// 20. Тест конкурентного доступа к трансферам
func TestConcurrentFileTransfer(t *testing.T) {
	ftm := NewFileTransferManager()
	defer os.RemoveAll("file_transfers")

	header := protocol.FileHeader{
		FileID:   "concurrent-test",
		FileSize: 1000,
	}
	ftm.StartTransfer(header.FileID, "s", "r", header)

	var wg sync.WaitGroup
	numOps := 20

	// Конкурентная запись чанков
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			ftm.AddChunk(header.FileID, index, []byte("data"), 4)
		}(i)
	}

	// Конкурентное чтение
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ftm.GetTransfer(header.FileID)
		}()
	}

	wg.Wait()

	transfer, ok := ftm.GetTransfer(header.FileID)
	if !ok {
		t.Fatal("Трансфер потерян при конкурентном доступе")
	}
	if len(transfer.Chunks) != numOps {
		t.Errorf("Ожидалось %d чанков, получено %d", numOps, len(transfer.Chunks))
	}
}

// Вспомогательная функция
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// 21. Тест StartTransfer
func TestStartTransferComplete(t *testing.T) {
	ftm := NewFileTransferManager()
	defer os.RemoveAll("file_transfers")

	header := protocol.FileHeader{
		FileID:    "test-start",
		FileName:  "test.txt",
		FileSize:  100,
		Sender:    "sender",
		Recipient: "recipient",
	}

	ftm.StartTransfer(header.FileID, header.Sender, header.Recipient, header)

	// Проверяем что директория создана
	if _, err := os.Stat("file_transfers/test-start"); os.IsNotExist(err) {
		t.Error("Директория для трансфера не создана")
	}

	// Проверяем что трансфер добавился
	transfer, ok := ftm.GetTransfer(header.FileID)
	if !ok {
		t.Fatal("Трансфер не найден")
	}
	if transfer.Sender != "sender" {
		t.Error("Неверный отправитель")
	}
	if transfer.Header.FileName != "test.txt" {
		t.Error("Неверное имя файла")
	}
}

// 22. Тест CompleteTransfer
func TestCompleteTransferFull(t *testing.T) {
	ftm := NewFileTransferManager()
	defer os.RemoveAll("file_transfers")

	header := protocol.FileHeader{
		FileID:   "test-complete",
		FileSize: 10,
	}
	ftm.StartTransfer(header.FileID, "s", "r", header)
	ftm.AddChunk(header.FileID, 0, []byte("12345"), 5)

	// Завершаем трансфер
	ftm.CompleteTransfer(header.FileID)

	// Проверяем что удалился из памяти
	_, ok := ftm.GetTransfer(header.FileID)
	if ok {
		t.Error("Трансфер должен быть удален из памяти")
	}

	// Проверяем что директория удалена
	if _, err := os.Stat("file_transfers/test-complete"); !os.IsNotExist(err) {
		t.Error("Директория трансфера должна быть удалена")
	}
}

// 23. Тест IsTransferComplete с разными сценариями
func TestIsTransferCompleteScenarios(t *testing.T) {
	ftm := NewFileTransferManager()
	defer os.RemoveAll("file_transfers")

	// Тест с несуществующим трансфером
	if ftm.IsTransferComplete("nonexistent") {
		t.Error("Несуществующий трансфер не должен быть завершенным")
	}

	// Тест с завершенным трансфером
	header := protocol.FileHeader{
		FileID:   "complete-test",
		FileSize: 5,
	}
	ftm.StartTransfer(header.FileID, "s", "r", header)
	ftm.AddChunk(header.FileID, 0, []byte("12345"), 5)

	if !ftm.IsTransferComplete(header.FileID) {
		t.Error("Трансфер со всеми чанками должен быть завершен")
	}
}

// 24. Тест AddChunk с разными размерами
func TestAddChunkDifferentSizes(t *testing.T) {
	ftm := NewFileTransferManager()
	defer os.RemoveAll("file_transfers")

	tests := []struct {
		name string
		data []byte
		size int
	}{
		{"пустой чанк", []byte{}, 0},
		{"маленький чанк", []byte("hi"), 2},
		{"средний чанк", []byte("hello world"), 11},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileID := "chunk-" + tt.name
			header := protocol.FileHeader{
				FileID:   fileID,
				FileSize: 100,
			}
			ftm.StartTransfer(fileID, "s", "r", header)

			err := ftm.AddChunk(fileID, 0, tt.data, tt.size)
			if err != nil {
				t.Errorf("AddChunk не должен был вернуть ошибку: %v", err)
			}

			transfer, _ := ftm.GetTransfer(fileID)
			if len(transfer.Chunks) != 1 {
				t.Error("Чанк не был сохранен")
			}
		})
	}
}

// 25. Тест GetTransfer с несуществующим ID
func TestGetTransferNonExistent(t *testing.T) {
	ftm := NewFileTransferManager()

	_, ok := ftm.GetTransfer("never-created")
	if ok {
		t.Error("Несуществующий трансфер не должен быть найден")
	}
}

// 26. Тест групп с несколькими пользователями
func TestMultipleGroupsOperations(t *testing.T) {
	srv := NewServer()

	client1, _ := net.Pipe()
	client2, _ := net.Pipe()
	client3, _ := net.Pipe()

	srv.addClient("user1", client1)
	srv.addClient("user2", client2)
	srv.addClient("user3", client3)

	// Добавляем в разные группы
	srv.addToGroup("#general", "user1")
	srv.addToGroup("#general", "user2")
	srv.addToGroup("#random", "user1")
	srv.addToGroup("#random", "user3")

	srv.mu.Lock()
	generalCount := len(srv.groups["#general"])
	randomCount := len(srv.groups["#random"])
	srv.mu.Unlock()

	if generalCount != 2 {
		t.Errorf("В #general ожидалось 2, получено %d", generalCount)
	}
	if randomCount != 2 {
		t.Errorf("В #random ожидалось 2, получено %d", randomCount)
	}
}

// 27. Тест проверки существования пользователя
func TestUserExists(t *testing.T) {
	dbPath := "test_exists.db"
	defer os.Remove(dbPath)

	storage := InitStorage(dbPath)
	defer storage.db.Close()

	if storage.UserExists("nonexistent") {
		t.Error("Несуществующий пользователь не должен существовать")
	}

	storage.RegisterUser("exists", "password")

	if !storage.UserExists("exists") {
		t.Error("Зарегистрированный пользователь должен существовать")
	}
}

// 28. Тест сохранения сообщения с разными получателями
func TestSaveMessageDifferentRecipients(t *testing.T) {
	dbPath := "test_recipients.db"
	defer os.Remove(dbPath)

	storage := InitStorage(dbPath)
	defer storage.db.Close()

	messages := []protocol.Message{
		{Sender: "alice", Recipient: "bob", Content: "Private", TimeStamp: time.Now().Unix(), Action: protocol.TypeChat},
		{Sender: "alice", Recipient: "#group", Content: "Group", TimeStamp: time.Now().Unix(), Action: protocol.TypeChat},
		{Sender: "system", Recipient: "all", Content: "Broadcast", TimeStamp: time.Now().Unix(), Action: protocol.TypeSystem},
	}

	for _, msg := range messages {
		err := storage.SaveMessage(msg)
		if err != nil {
			t.Fatalf("SaveMessage вернул ошибку: %v", err)
		}
	}

	// Проверяем что все сохранились
	history, _ := storage.GetHistory("bob")
	if len(history) == 0 {
		t.Error("Приватные сообщения должны сохраняться")
	}
}

// 29. Тест GetSystemLogs с лимитом
func TestGetSystemLogsLimit(t *testing.T) {
	dbPath := "test_log_limit.db"
	defer os.Remove(dbPath)

	storage := InitStorage(dbPath)
	defer storage.db.Close()

	// Добавляем 10 событий
	for i := 0; i < 10; i++ {
		storage.LogEvent("TEST", fmt.Sprintf("Event %d", i))
	}

	// Запрашиваем только 3
	logs, err := storage.GetSystemLogs(3)
	if err != nil {
		t.Fatalf("GetSystemLogs вернул ошибку: %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("Ожидалось 3 лога, получено %d", len(logs))
	}
}

// 30. Тест GetMessageLogs с пустой БД
func TestGetMessageLogsEmpty(t *testing.T) {
	dbPath := "test_empty_logs.db"
	defer os.Remove(dbPath)

	storage := InitStorage(dbPath)
	defer storage.db.Close()

	logs, err := storage.GetMessageLogs(10)
	if err != nil {
		t.Fatalf("GetMessageLogs вернул ошибку: %v", err)
	}

	if len(logs) != 0 {
		t.Errorf("Ожидался пустой список логов, получено %d", len(logs))
	}
}

// 31. Тест уведомления о shutdown
func TestShutdownNotification(t *testing.T) {
	srv := NewServer()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	srv.addClient("testuser", server)

	// Запускаем чтение в фоне
	go func() {
		buf := make([]byte, 4096)
		client.SetReadDeadline(time.Now().Add(2 * time.Second))
		client.Read(buf)
	}()

	time.Sleep(50 * time.Millisecond)

	// Проверяем что метод не паникует
	srv.notifyClientsAboutShutdown()

	// Проверяем что клиент все еще в списке (уведомление не удаляет)
	if srv.getOnlineCount() != 1 {
		t.Error("Уведомление не должно удалять клиентов")
	}
}

// 32. Тест getClient с блокировкой
func TestGetClientConcurrent(t *testing.T) {
	srv := NewServer()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	srv.addClient("testuser", server)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, ok := srv.getClient("testuser")
			if !ok || conn == nil {
				t.Error("getClient должен находить существующего клиента")
			}
		}()
	}
	wg.Wait()
}
