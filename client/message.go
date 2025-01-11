package client

import (
	"fmt"
)

// Messages can be
// recevied by a server, this server typically sends the raw data in the correct format
// sent by the client, we need to enrich a user input with author and timestamp before sending it.

type Message struct {
	Author    string    `json:"author"`
	Timestamp string    `json:"timestamp"`
	Body      string    `json:"body"`
}

func (m Message) String() string {
    return fmt.Sprintf("%s - %s: %s\n", m.Timestamp, m.Author, m.Body)
}
