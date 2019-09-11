package collector

import (
	"encoding/json"
	"log"
	"sync"

	"gopkg.in/olahol/melody.v1"
)

// statsMessage struct of message, received from ws client
type statsMessage struct {
	ID    uint64 `json:"id"`
	Label string `json:"label"`
}

func (m *statsMessage) reset() {
	m.ID = 0
	m.Label = ""
}

// GetMessageHandler returns handler for incoming ws messages
//
// Unmarshal incoming message into `statsMessage` and call `Storage.Add` for received data
func GetMessageHandler(storage *Storage) func(*melody.Session, []byte) {
	pool := sync.Pool{
		New: func() interface{} { return new(statsMessage) },
	}

	return func(_ *melody.Session, msg []byte) {
		message := pool.Get().(*statsMessage)
		if err := json.Unmarshal(msg, message); err != nil {
			log.Printf("failed to unmarshal incoming message: %s", err.Error())
			return
		}
		storage.Add(message.ID, message.Label, 1)

		message.reset()
		pool.Put(message)
	}
}
