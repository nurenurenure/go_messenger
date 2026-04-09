package protocol

import (
	"encoding/binary"
	"errors"
	"io"
)

const (
	// MagicByte — сигнатура нашего мессенджера (буква 'M' в шестнадцатеричной системе).
	// Помогает сразу отбрасывать левый трафик.
	MagicByte byte = 0x4D

	// ProtocolVer — версия протокола. (пригодится при смене протокола)
	ProtocolVer byte = 0x01

	// HeaderSize — фиксированный размер нашего заголовка (8 байт).
	// 1 (Magic) + 1 (Ver) + 2 (Type) + 4 (Length) = 8 байт.
	HeaderSize int = 8
)

// Ошибки протокола
var (
	ErrInvalidMagic   = errors.New("invalid magic byte: unrecognized protocol")
	ErrInvalidVersion = errors.New("unsupported protocol version")
)

// WritePacket берет тип сообщения и его тело (через Protobuf),
// формирует бинарный заголовок и отправляет всё это в сетевое соединение.
func WritePacket(w io.Writer, msgType uint16, payload []byte) error {
	// 1. Создаем буфер для заголовка строго на 8 байт
	header := make([]byte, HeaderSize)

	// 2. Заполняем первые два байта вручную
	header[0] = MagicByte
	header[1] = ProtocolVer

	// 3. Записываем тип сообщения (2 байта) с использованием BigEndian (сетевой стандарт)
	binary.BigEndian.PutUint16(header[2:4], msgType)

	// 4. Записываем длину тела (4 байта)
	// Используем uint32, что позволяет отправлять сообщения размером до 4 ГБ
	binary.BigEndian.PutUint32(header[4:8], uint32(len(payload)))

	// 5. Отправляем заголовок в сеть
	if _, err := w.Write(header); err != nil {
		return err
	}

	// 6. Если есть тело сообщения, отправляем и его
	if len(payload) > 0 {
		if _, err := w.Write(payload); err != nil {
			return err
		}
	}

	return nil
}

// ReadPacket читает данные из сетевого соединения, сначала разбирая заголовок,
// а затем вычитывая ровно столько байт тела, сколько указано в заголовке.
func ReadPacket(r io.Reader) (uint16, []byte, error) {
	// 1. Читаем ровно 8 байт заголовка.
	// io.ReadFull гарантирует, что мы не получим обрывок заголовка (например, только 3 байта).
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return 0, nil, err
	}

	// 2. Проверяем валидность пакета
	if header[0] != MagicByte {
		return 0, nil, ErrInvalidMagic
	}
	if header[1] != ProtocolVer {
		return 0, nil, ErrInvalidVersion
	}

	// 3. Достаем тип сообщения и длину тела
	msgType := binary.BigEndian.Uint16(header[2:4])
	length := binary.BigEndian.Uint32(header[4:8])

	var payload []byte

	// 4. Если длина больше нуля, читаем тело сообщения
	if length > 0 {
		payload = make([]byte, length)
		// Снова используем io.ReadFull, чтобы дождаться полного получения всех данных тела
		if _, err := io.ReadFull(r, payload); err != nil {
			return 0, nil, err
		}
	}

	// Возвращаем извлеченный тип и само сообщение (которое потом передадим в Protobuf Unmarshal)
	return msgType, payload, nil
}
