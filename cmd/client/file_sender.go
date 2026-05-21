package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"messenger/protocol"
)

type FileSender struct {
	conn     net.Conn
	username string
}

func NewFileSender(conn net.Conn, username string) *FileSender {
	return &FileSender{
		conn:     conn,
		username: username,
	}
}

func (fs *FileSender) SendFileRequest(recipient, filePath string) (*protocol.FileHeader, error) {
	//Проверка существования файла
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("файл не найден: %w", err)
	}
	//Вычисление контрольной суммы
	checksum, err := protocol.CalculateFileChecksum(filePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка вычисления контрольной суммы: %w", err)
	}

	//Извлечение имени файла
	fileName := filepath.Base(filePath)
	//Генерация FileID
	fileID := protocol.GenerateFileID(fs.username, recipient, fileName)

	header := &protocol.FileHeader{
		FileName:  fileName,
		FileSize:  fileInfo.Size(),
		Checksum:  checksum,
		Sender:    fs.username,
		Recipient: recipient,
		FileID:    fileID,
	}

	err = protocol.WritePacket(fs.conn, protocol.TypeFileRequest, *header)
	if err != nil {
		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
	}

	//Заголовок создаётся один раз и передаётся между функциями
	return header, nil
}

//header protocol.FileHeader — заголовок, созданный в SendFileRequest (по значению, не по указателю — ок, структура маленькая).

//filePath string — путь к файлу (повторно открывается).

//progressChan chan<- float64 — однонаправленный канал только для отправки.

func (fs *FileSender) SendFile(header protocol.FileHeader, filePath string, progressChan chan<- float64) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()

	if err := protocol.WritePacket(fs.conn, protocol.TypeFileHeader, header); err != nil {
		return fmt.Errorf("ошибка отправки заголовка: %w", err)
	}

	//Вычисление количества чанков
	totalChunks := int(header.FileSize / protocol.ChunkSize)
	if header.FileSize%protocol.ChunkSize != 0 {
		totalChunks++
	}

	//Цикл отправки чанков
	buffer := make([]byte, protocol.ChunkSize) //переиспользуется для каждого чанка
	for i := 0; i < totalChunks; i++ {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("ошибка чтения файла: %w", err)
		}

		chunk := protocol.FileChunk{
			FileID:      header.FileID,
			ChunkIndex:  i,
			TotalChunks: totalChunks,
			Data:        buffer[:n],
			Size:        n,
		}

		if err := protocol.WritePacket(fs.conn, protocol.TypeFileChunk, chunk); err != nil {
			return fmt.Errorf("ошибка отправки чанка %d: %w", i, err)
		}
		//канал прогресса
		if progressChan != nil {
			progressChan <- float64(i+1) / float64(totalChunks) * 100
		}

		time.Sleep(10 * time.Millisecond)

		if n < protocol.ChunkSize {
			break
		}
	}

	return nil
}
