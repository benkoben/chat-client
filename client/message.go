package client

import (
	"fmt"
)

// Messages can be
// recevied by a server, this server typically sends the raw data in the correct format
// sent by the client, we need to enrich a user input with author and timestamp before sending it.
const (
    msgTypeHello messageType   = iota 
    msgTypeMessage messageType = iota 
    msgTypeBye messageType     = iota 
)

type messageType int

func (m messageType)String()string{
    var mType string
    switch {
        case m == 0:
            mType = "hello"
        case m == 1:
            mType = "message"
        case m == 2:
            mType = "bye"
    }

    return mType
}

type Message struct {
	Author    string      `json:"author"`
	Timestamp string      `json:"timestamp"`
	Body      string      `json:"body"`
	Type      messageType `json:"type"`
}

// Formats a message into a readable format. Omits the type field in the return value
func (m Message) String() string {
    return fmt.Sprintf("%s - %s: %s\n", m.Timestamp, m.Author, m.Body)
}
