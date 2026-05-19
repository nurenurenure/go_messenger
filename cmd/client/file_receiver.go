package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"messenger/protocol"
)

type FileReceiver struct {
	conn        net.Conn
	username    string
	downloadDir string
	activeFiles map[string]*ReceivingFile
	onComplete  func(fileName string, filePath string)
}

type ReceivingFile struct {
	Header         protocol.FileHeader
	Chunks         [][]byte
	ReceivedChunks int
	TotalChunks    int
	StartTime      time.Time
}

func NewFileReceiver(conn net.Conn, username string) *FileReceiver {
	downloadDir := filepath.Join("downloads", username)
	os.MkdirAll(downloadDir, 0755)

	return &FileReceiver{
		conn:        conn,
		username:    username,
		downloadDir: downloadDir,
		activeFiles: make(map[string]*ReceivingFile),
	}
}

func (fr *FileReceiver) SetOnComplete(callback func(fileName string, filePath string)) {
	fr.onComplete = callback
}

func (fr *FileReceiver) HandleFileMessage(msgType uint8, payload []byte) {
	switch msgType {
	case protocol.TypeFileRequest:
		fr.handleFileRequest(payload)
	case protocol.TypeFileHeader:
		fr.handleFileHeader(payload)
	case protocol.TypeFileChunk:
		fr.handleFileChunk(payload)
	case protocol.TypeFileComplete:
		fr.handleFileComplete(payload)
	case protocol.TypeFileError:
		fr.handleFileError(payload)
	}
}

func (fr *FileReceiver) handleFileRequest(payload []byte) {
	var header protocol.FileHeader
	if err := json.Unmarshal(payload, &header); err != nil {
		return
	}

	fr.AcceptFile(header)
}

func (fr *FileReceiver) AcceptFile(header protocol.FileHeader) {
	protocol.WritePacket(fr.conn, protocol.TypeFileAccept, header)
	fmt.Printf("Принимаем файл: %s (%d байт)\n", header.FileName, header.FileSize)
}

func (fr *FileReceiver) RejectFile(header protocol.FileHeader) {
	protocol.WritePacket(fr.conn, protocol.TypeFileReject, header)
}

func (fr *FileReceiver) handleFileHeader(payload []byte) {
	var header protocol.FileHeader
	if err := json.Unmarshal(payload, &header); err != nil {
		return
	}

	totalChunks := int(header.FileSize / protocol.ChunkSize)
	if header.FileSize%protocol.ChunkSize != 0 {
		totalChunks++
	}

	fr.activeFiles[header.FileID] = &ReceivingFile{
		Header:      header,
		Chunks:      make([][]byte, totalChunks),
		StartTime:   time.Now(),
		TotalChunks: totalChunks,
	}

	fmt.Printf("Получение файла: %s (%d чанков)\n", header.FileName, totalChunks)
}

func (fr *FileReceiver) handleFileChunk(payload []byte) {
	var chunk protocol.FileChunk
	if err := json.Unmarshal(payload, &chunk); err != nil {
		return
	}

	receiving, ok := fr.activeFiles[chunk.FileID]
	if !ok {
		return
	}

	data := make([]byte, chunk.Size)
	copy(data, chunk.Data[:chunk.Size])
	receiving.Chunks[chunk.ChunkIndex] = data
	receiving.ReceivedChunks++

	// Всегда выводим прогресс
	progress := float64(receiving.ReceivedChunks) / float64(receiving.TotalChunks) * 100
	fmt.Print("\r" + strings.Repeat(" ", 80) + "\r")
	fmt.Printf("Прогресс: %.1f%% (%d/%d)",
		progress, receiving.ReceivedChunks, receiving.TotalChunks)

	// Проверяем завершение
	if receiving.ReceivedChunks >= receiving.TotalChunks {
		// Очищаем строку прогресса
		fmt.Print("\r" + strings.Repeat(" ", 80) + "\r")
		fmt.Println() // Перевод строки
		fr.assembleFile(chunk.FileID)
	}
}
func (fr *FileReceiver) assembleFile(fileID string) {
	receiving, ok := fr.activeFiles[fileID]
	if !ok {
		return
	}

	var fileData []byte
	for i := 0; i < receiving.TotalChunks; i++ {
		if receiving.Chunks[i] == nil {
			fmt.Printf("Ошибка: отсутствует чанк %d\n", i)
			return
		}
		fileData = append(fileData, receiving.Chunks[i]...)
	}

	actualChecksum := protocol.CalculateChecksumFromBytes(fileData)

	complete := protocol.FileComplete{
		FileID:   fileID,
		Checksum: actualChecksum,
	}

	if actualChecksum == receiving.Header.Checksum {
		filePath := filepath.Join(fr.downloadDir, receiving.Header.FileName)
		if err := os.WriteFile(filePath, fileData, 0644); err != nil {
			complete.Status = "error"
			protocol.WritePacket(fr.conn, protocol.TypeFileComplete, complete)
			fmt.Printf("Ошибка сохранения файла: %v\n", err)
			return
		}

		complete.Status = "ok"
		protocol.WritePacket(fr.conn, protocol.TypeFileComplete, complete)

		elapsed := time.Since(receiving.StartTime)
		speed := float64(len(fileData)) / elapsed.Seconds() / 1024

		// Чистый вывод без мусора
		fmt.Printf("Файл получен: %s (%d байт, %.1f KB/s)\n",
			receiving.Header.FileName, len(fileData), speed)
		if fr.onComplete != nil {
			fr.onComplete(receiving.Header.FileName, filePath)
		}
	} else {
		complete.Status = "error"
		protocol.WritePacket(fr.conn, protocol.TypeFileComplete, complete)
		fmt.Printf("Ошибка: контрольная сумма не совпадает!\n")
	}

	delete(fr.activeFiles, fileID)
}

func (fr *FileReceiver) handleFileComplete(payload []byte) {
	var complete protocol.FileComplete
	if err := json.Unmarshal(payload, &complete); err != nil {
		return
	}

	if complete.Status != "ok" {
		fmt.Println("Ошибка при передаче файла")
	}
}

func (fr *FileReceiver) handleFileError(payload []byte) {
	var errorMsg protocol.Message
	if err := json.Unmarshal(payload, &errorMsg); err != nil {
		return
	}

	fmt.Printf("Ошибка передачи файла: %s\n", errorMsg.Content)
}
