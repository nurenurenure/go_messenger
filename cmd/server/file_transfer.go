package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"messenger/protocol"
)

type FileTransfer struct {
	Header    protocol.FileHeader
	Chunks    map[int][]byte
	Sender    string
	Recipient string
}

type FileTransferManager struct {
	transfers map[string]*FileTransfer
	mu        sync.RWMutex
}

func NewFileTransferManager() *FileTransferManager {
	return &FileTransferManager{
		transfers: make(map[string]*FileTransfer),
	}
}

func (ftm *FileTransferManager) StartTransfer(fileID, sender, recipient string, header protocol.FileHeader) {
	ftm.mu.Lock()
	defer ftm.mu.Unlock()

	ftm.transfers[fileID] = &FileTransfer{
		Header:    header,
		Chunks:    make(map[int][]byte),
		Sender:    sender,
		Recipient: recipient,
	}

	tempDir := filepath.Join("file_transfers", fileID)
	os.MkdirAll(tempDir, 0755)
}

func (ftm *FileTransferManager) AddChunk(fileID string, chunkIndex int, data []byte, size int) error {
	ftm.mu.Lock()
	defer ftm.mu.Unlock()

	transfer, ok := ftm.transfers[fileID]
	if !ok {
		return fmt.Errorf("передача не найдена: %s", fileID)
	}

	transfer.Chunks[chunkIndex] = make([]byte, size)
	copy(transfer.Chunks[chunkIndex], data[:size])

	chunkPath := filepath.Join("file_transfers", fileID, fmt.Sprintf("chunk_%d", chunkIndex))
	return os.WriteFile(chunkPath, data[:size], 0644)
}

func (ftm *FileTransferManager) GetTransfer(fileID string) (*FileTransfer, bool) {
	ftm.mu.RLock()
	defer ftm.mu.RUnlock()

	transfer, ok := ftm.transfers[fileID]
	return transfer, ok
}

func (ftm *FileTransferManager) IsTransferComplete(fileID string) bool {
	ftm.mu.RLock()
	defer ftm.mu.RUnlock()

	transfer, ok := ftm.transfers[fileID]
	if !ok {
		return false
	}

	totalChunks := int(transfer.Header.FileSize / protocol.ChunkSize)
	if transfer.Header.FileSize%protocol.ChunkSize != 0 {
		totalChunks++
	}

	return len(transfer.Chunks) >= totalChunks
}

func (ftm *FileTransferManager) GetAssembledFile(fileID string) ([]byte, error) {
	ftm.mu.RLock()
	defer ftm.mu.RUnlock()

	transfer, ok := ftm.transfers[fileID]
	if !ok {
		return nil, fmt.Errorf("передача не найдена: %s", fileID)
	}

	totalChunks := int(transfer.Header.FileSize / protocol.ChunkSize)
	if transfer.Header.FileSize%protocol.ChunkSize != 0 {
		totalChunks++
	}

	var fileData []byte
	for i := 0; i < totalChunks; i++ {
		chunk, exists := transfer.Chunks[i]
		if !exists {
			return nil, fmt.Errorf("отсутствует чанк %d", i)
		}
		fileData = append(fileData, chunk...)
	}

	return fileData, nil
}

func (ftm *FileTransferManager) CompleteTransfer(fileID string) {
	ftm.mu.Lock()
	defer ftm.mu.Unlock()

	delete(ftm.transfers, fileID)
	tempDir := filepath.Join("file_transfers", fileID)
	os.RemoveAll(tempDir)
}
