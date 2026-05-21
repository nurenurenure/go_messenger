package protocol

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
)

// константы типов сообщений
const (
	TypeAuth   uint8 = 0x01
	TypeChat   uint8 = 0x02
	TypeStatus uint8 = 0x03
	TypeSystem uint8 = 0x04
	TypeError  uint8 = 0x05
)

type Message struct {
	ID        string `json:"id"`
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Content   string `json:"content"`
	TimeStamp int64  `json:"timestamp"`
	Action    uint8  `json:"action"`
}

// отправка пакета
func WritePacket(conn net.Conn, msgType uint8, data interface{}) error {
	//JSON-сериализация
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	//подготовить заголовок(4 байта на длину, 1 байт на тип сообщения)
	header := make([]byte, 5)
	binary.BigEndian.PutUint32(header[:4], uint32(len(payload)))
	header[4] = msgType
	//отправить заголовок и тело сообщения в соединение
	_, err = conn.Write(append(header, payload...))
	return err
}

// чтение пакета
func ReadPacket(conn net.Conn) (uint8, []byte, error) {
	//прочитать заголовок
	header := make([]byte, 5)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		return 0, nil, err
	}
	//извлечь длину и тип сообщения
	length := binary.BigEndian.Uint32(header[:4])
	msgType := header[4]
	//извлечь тело сообщения из соединения
	payload := make([]byte, length)
	_, err = io.ReadFull(conn, payload)
	if err != nil {
		return 0, nil, err
	}
	return msgType, payload, nil

}
