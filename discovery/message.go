package discovery

import (
	"encoding/base64"
	"encoding/json"
	"time"
)

type Message struct {
	Type       string `json:"type"`
	SenderId   string `json:"sender_id"`
	ReceiverId string `json:"receiver_id"`
	Payload    []byte `json:"payload"`
	Timestamp  int64  `json:"timestamp"`
	IsBinary   bool   `json:"is_binary"`
}

func NewMessage(msgType, sender, receiver string, payload []byte, isBinary bool) Message {
	return Message{
		Type:       msgType,
		SenderId:   sender,
		ReceiverId: receiver,
		Payload:    payload,
		Timestamp:  time.Now().Unix(),
		IsBinary:   isBinary,
	}
}

// MarshalJSON ensures correct encoding of binary data.
func (m *Message) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type       string `json:"type"`
		SenderID   string `json:"sender_id"`
		ReceiverID string `json:"receiver_id"`
		Payload    string `json:"payload"` // Encode as base64 string for safety
		Timestamp  int64  `json:"timestamp"`
		IsBinary   bool   `json:"is_binary"`
	}{
		Type:       m.Type,
		SenderID:   m.SenderId,
		ReceiverID: m.ReceiverId,
		Payload:    encodeBase64(m.Payload),
		Timestamp:  m.Timestamp,
		IsBinary:   m.IsBinary,
	})
}

// UnmarshalJSON ensures correct decoding of binary data.
func (m *Message) UnmarshalJSON(data []byte) error {
	aux := struct {
		Type       string `json:"type"`
		SenderID   string `json:"sender_id"`
		ReceiverID string `json:"receiver_id"`
		Payload    string `json:"payload"`
		Timestamp  int64  `json:"timestamp"`
		IsBinary   bool   `json:"is_binary"`
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	m.Type = aux.Type
	m.SenderId = aux.SenderID
	m.ReceiverId = aux.ReceiverID
	m.Timestamp = aux.Timestamp
	m.IsBinary = aux.IsBinary
	m.Payload, _ = decodeBase64(aux.Payload)
	return nil
}

// encodeBase64 converts a byte slice to a base64-encoded string.
func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// decodeBase64 converts a base64-encoded string back into a byte slice.
func decodeBase64(encoded string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err // Returning nil in case of an error
	}
	return data, nil
}
