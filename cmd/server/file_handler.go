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
		s.handleFileAccept(payload)
	case protocol.TypeFileReject:
		s.handleFileReject(payload)
	case protocol.TypeFileHeader:
		s.handleFileHeader(payload, username)
	case protocol.TypeFileChunk:
		s.handleFileChunk(payload)
	case protocol.TypeFileComplete:
		s.handleFileComplete(payload)
	}
}

func (s *Server) handleFileRequest(payload []byte, sender string) {
	var header protocol.FileHeader
	if err := json.Unmarshal(payload, &header); err != nil {
		return
	}

	recipientConn, ok := s.getClient(header.Recipient)
	if !ok {
		// Отправляем ошибку отправителю
		errorMsg := protocol.Message{
			Sender:    "Система",
			Recipient: sender,
			Content:   fmt.Sprintf("Пользователь %s не в сети", header.Recipient),
			Action:    protocol.TypeChat,
		}
		if conn, ok := s.getClient(sender); ok {
			protocol.WritePacket(conn, protocol.TypeChat, errorMsg)
		}
		return
	}

	protocol.WritePacket(recipientConn, protocol.TypeFileRequest, header)
}

func (s *Server) handleFileAccept(payload []byte) {
	var header protocol.FileHeader
	if err := json.Unmarshal(payload, &header); err != nil {
		return
	}

	senderConn, ok := s.getClient(header.Sender)
	if ok {
		protocol.WritePacket(senderConn, protocol.TypeFileAccept, header)
	}
}

func (s *Server) handleFileReject(payload []byte) {
	var header protocol.FileHeader
	if err := json.Unmarshal(payload, &header); err != nil {
		return
	}

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

	// Сохраняем информацию о передаче
	s.fileTransfers.StartTransfer(header.FileID, sender, header.Recipient, header)

	// Пересылаем заголовок получателю
	recipientConn, ok := s.getClient(header.Recipient)
	if ok {
		protocol.WritePacket(recipientConn, protocol.TypeFileHeader, header)
	}
}

func (s *Server) handleFileChunk(payload []byte) {
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
}

func (s *Server) handleFileComplete(payload []byte) {
	var complete protocol.FileComplete
	if err := json.Unmarshal(payload, &complete); err != nil {
		return
	}

	transfer, ok := s.fileTransfers.GetTransfer(complete.FileID)
	if !ok {
		return
	}

	// Пересылаем подтверждение отправителю
	senderConn, ok := s.getClient(transfer.Sender)
	if ok {
		protocol.WritePacket(senderConn, protocol.TypeFileComplete, complete)
	}

	// Очищаем передачу
	s.fileTransfers.CompleteTransfer(complete.FileID)
}
