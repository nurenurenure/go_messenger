package protocol

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	// Типы для передачи файлов
	TypeFileRequest  uint8 = 0x10
	TypeFileAccept   uint8 = 0x11
	TypeFileReject   uint8 = 0x12
	TypeFileHeader   uint8 = 0x13
	TypeFileChunk    uint8 = 0x14
	TypeFileComplete uint8 = 0x15
	TypeFileError    uint8 = 0x16
)
const ChunkSize = 4096 // 4KB размер чанка

type FileHeader struct {
	FileName  string `json:"file_name"`
	FileSize  int64  `json:"file_size"`
	Checksum  string `json:"checksum"` //SHA-256 всего файла
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	FileID    string `json:"file_id"`
}

type FileChunk struct {
	FileID      string `json:"file_id"` //связывает чанк с конкретным файлом
	ChunkIndex  int    `json:"chunk_index"`
	TotalChunks int    `json:"total_chunks"`
	Data        []byte `json:"data"`
	Size        int    `json:"size"`
}

type FileComplete struct {
	FileID   string `json:"file_id"`
	Checksum string `json:"checksum"`
	Status   string `json:"status"`
}

func CalculateFileChecksum(filePath string) (string, error) {
	//открытие файла на чтение
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()
	//создание объекта hash.Hash
	hash := sha256.New()
	//потоковоя передача в хеш
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("ошибка вычисления хеша: %w", err)
	}
	//возвращаем 32-байтовый массив (SHA-256)
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func CalculateChecksumFromBytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// генерирация уникального ID файла
func GenerateFileID(sender, recipient, fileName string) string {
	data := fmt.Sprintf("%s-%s-%s-%d", sender, recipient, fileName, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:16] //Укороченный ID легче логировать и передавать
}
