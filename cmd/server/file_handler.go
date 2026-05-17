package main

import (
	"encoding/json"
	"fmt"

	"messenger/protocol"
)

func (s *Server) handleFileTransfer(msgType uint8, payload []byte, username string) {
	switch msgType {
	case protocol.TypeFileRequest:
		s.handleFileRequest(payload, username)
	case protocol.TypeFileAccept:
		s.handleFileAccept(payload, username)
	case protocol.TypeFileReject:
		s.handleFileReject(payload, username)
	case protocol.TypeFileHeader:
		s.handleFileHeader(payload, username)
	case protocol.TypeFileChunk:
		s.handleFileChunk(payload, username)
	case protocol.TypeFileComplete:
		s.handleFileComplete(payload, username)
	}
}

func (s *Server) handleFileRequest(payload []byte, sender string) {
	var header protocol.FileHeader
	if err := json.Unmarshal(payload, &header); err != nil {
		return
	}

	// Логируем запрос
	s.storage.LogFileEvent("FILE_REQUEST", sender, header.Recipient,
		header.FileName, header.FileSize, header.FileID)

	recipientConn, ok := s.getClient(header.Recipient)
	if !ok {
		fmt.Printf("Получатель %s не в сети для файла от %s\n", header.Recipient, sender)
		return
	}

	protocol.WritePacket(recipientConn, protocol.TypeFileRequest, header)
}

func (s *Server) handleFileAccept(payload []byte, _ string) {
	var header protocol.FileHeader
	if err := json.Unmarshal(payload, &header); err != nil {
		return
	}

	// Логируем принятие
	s.storage.LogFileEvent("FILE_ACCEPT", header.Recipient, header.Sender,
		header.FileName, header.FileSize, header.FileID)

	senderConn, ok := s.getClient(header.Sender)
	if ok {
		protocol.WritePacket(senderConn, protocol.TypeFileAccept, header)
	}
}

func (s *Server) handleFileReject(payload []byte, _ string) {
	var header protocol.FileHeader
	if err := json.Unmarshal(payload, &header); err != nil {
		return
	}

	// Логируем отказ
	s.storage.LogFileEvent("FILE_REJECT", header.Recipient, header.Sender,
		header.FileName, header.FileSize, header.FileID)

	senderConn, ok := s.getClient(header.Sender)
	if ok {
		protocol.WritePacket(senderConn, protocol.TypeFileReject, header)
	}
}

func (s *Server) handleFileHeader(payload []byte, sender string) {
	var header protocol.FileHeader
	if err := json.Unmarshal(payload, &header); err != nil {
		return
	}

	// Логируем начало передачи
	s.storage.LogFileEvent("FILE_START", sender, header.Recipient,
		header.FileName, header.FileSize, header.FileID)

	// Сохраняем информацию о передаче
	s.fileTransfers.StartTransfer(header.FileID, sender, header.Recipient, header)

	recipientConn, ok := s.getClient(header.Recipient)
	if ok {
		protocol.WritePacket(recipientConn, protocol.TypeFileHeader, header)
	}
}

func (s *Server) handleFileChunk(payload []byte, _ string) {
	var chunk protocol.FileChunk
	if err := json.Unmarshal(payload, &chunk); err != nil {
		return
	}

	// Сохраняем чанк
	if err := s.fileTransfers.AddChunk(chunk.FileID, chunk.ChunkIndex, chunk.Data, chunk.Size); err != nil {
		fmt.Printf("Ошибка сохранения чанка: %v\n", err)
		return
	}

	// Пересылаем чанк получателю
	transfer, ok := s.fileTransfers.GetTransfer(chunk.FileID)
	if !ok {
		return
	}

	recipientConn, ok := s.getClient(transfer.Recipient)
	if ok {
		protocol.WritePacket(recipientConn, protocol.TypeFileChunk, chunk)
	}

	// Логируем прогресс каждые 10% чанков
	if chunk.ChunkIndex%max(1, chunk.TotalChunks/10) == 0 {
		progress := float64(chunk.ChunkIndex+1) / float64(chunk.TotalChunks) * 100
		s.storage.LogFileEvent("FILE_PROGRESS", transfer.Sender, transfer.Recipient,
			fmt.Sprintf("%s (%.1f%%)", transfer.Header.FileName, progress),
			transfer.Header.FileSize, chunk.FileID)
	}
}

func (s *Server) handleFileComplete(payload []byte, _ string) {
	var complete protocol.FileComplete
	if err := json.Unmarshal(payload, &complete); err != nil {
		return
	}

	transfer, ok := s.fileTransfers.GetTransfer(complete.FileID)
	if !ok {
		return
	}

	// Логируем завершение
	if complete.Status == "ok" {
		s.storage.LogFileEvent("FILE_COMPLETE", transfer.Sender, transfer.Recipient,
			transfer.Header.FileName, transfer.Header.FileSize, complete.FileID)
	} else {
		s.storage.LogFileEvent("FILE_ERROR", transfer.Sender, transfer.Recipient,
			transfer.Header.FileName+" (checksum mismatch)", transfer.Header.FileSize, complete.FileID)
	}

	// Пересылаем подтверждение отправителю
	senderConn, ok := s.getClient(transfer.Sender)
	if ok {
		protocol.WritePacket(senderConn, protocol.TypeFileComplete, complete)
	}

	// Очищаем передачу
	s.fileTransfers.CompleteTransfer(complete.FileID)
}
