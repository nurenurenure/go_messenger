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
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("файл не найден: %w", err)
	}

	checksum, err := protocol.CalculateFileChecksum(filePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка вычисления контрольной суммы: %w", err)
	}

	fileName := filepath.Base(filePath)
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

	return header, nil
}

func (fs *FileSender) SendFile(header protocol.FileHeader, filePath string, progressChan chan<- float64) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()

	if err := protocol.WritePacket(fs.conn, protocol.TypeFileHeader, header); err != nil {
		return fmt.Errorf("ошибка отправки заголовка: %w", err)
	}

	totalChunks := int(header.FileSize / protocol.ChunkSize)
	if header.FileSize%protocol.ChunkSize != 0 {
		totalChunks++
	}

	buffer := make([]byte, protocol.ChunkSize)
	for i := 0; i < totalChunks; i++ {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("ошибка чтения файла: %w", err)
		}

		chunk := protocol.FileChunk{
			FileID:      header.FileID,
			ChunkIndex:  i,
			TotalChunks: totalChunks,
			Data:        buffer,
			Size:        n,
		}

		if err := protocol.WritePacket(fs.conn, protocol.TypeFileChunk, chunk); err != nil {
			return fmt.Errorf("ошибка отправки чанка %d: %w", i, err)
		}

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
