package client

import (
	"encoding/json"
	"fmt"
	"time"
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

/*
messageBus is a buffered channel used by workers to receive and process messages
*/
type messageBus chan *Message

func newMessageBus(size int) *messageBus {
    if size == 0 {
        // Ensure that the messageBus is always a buffered channel
        size = 1 
    }

    var mb messageBus = make(chan *Message, size)
    return &mb
}

func (m messageType)String()string{
    var mType string
    switch {
        case m == 0:
            mType = "hello"
        case m == 1:
            mType = "message"
        case m == 2:
            mType = "bye"
        default:
            mType = "unknown"
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

// Just a simple wrapper method around json.Marshall
func (m Message) Bytes() ([]byte, error) {
    return json.Marshal(m)
}

// returns a raw byte slice of an Hello type message
func newRawHelloMsg(author string) []byte {
    m := Message{
        Timestamp: time.Now().Format(time.RFC850),
        Author: author,
        Type: msgTypeHello,
    }
    rawMsg, _ := json.Marshal(m)
    return rawMsg
}

// returns a raw byte slice of an bye type message
func newRawSystemMsg(author string, t messageType) ([]byte, error) {
    // TODO: Add validators here

    m := Message{
        Timestamp: time.Now().Format(time.RFC850),
        Author: author,
        Type: t,
    }

    rawMsg, _ := json.Marshal(m)
    return rawMsg, nil
}

func newRawMsg(name, body string) []byte{
    m := Message{
        Author: name,
        Body: body,
        Timestamp: time.Now().Format(time.RFC850),
        Type: msgTypeMessage,
    }
    rawMsg, _ := json.Marshal(m)
    return rawMsg
}

func newMsg(name, body string) *Message {
    return &Message{
        Author: name,
        Body: body,
        Timestamp: time.Now().Format(time.RFC850),
        Type: msgTypeMessage,
    }
}

func unmarshalMessage(data []byte) (*Message, error) {
   var msg Message
   err := json.Unmarshal(data, &msg) 
   if err != nil {
       return nil, fmt.Errorf("message: could not unmarshal data", err)
   }

   return &msg, nil
}

func isHello(in []byte) (bool, messageType) {
    var m Message
    if err := json.Unmarshal(in, &m); err != nil {
        return false, m.Type
    }

    if m.Type == msgTypeHello {
        return true, m.Type
    }

    return false, m.Type
}

