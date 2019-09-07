package collector

import (
	"encoding/json"
	"log"

	"gopkg.in/olahol/melody.v1"
)

// StatsMessage struct of message, received from ws client
type StatsMessage struct {
	Id    uint64 `json:"id"`
	Label string `json:"label"`
}

// GetMessageHandler returns handler for incoming ws messages
//
// Unmarshal incoming message into `StatsMessage` and call `Storage.Add` for received data
func GetMessageHandler(storage *Storage) func(*melody.Session, []byte) {
	return func(_ *melody.Session, msg []byte) {
		message := new(StatsMessage)
		if err := json.Unmarshal(msg, message); err != nil {
			log.Printf("failed to unmarshal incoming message: %s", err.Error())
			return
		}
		storage.Add(message.Id, message.Label, 1)
	}
}
