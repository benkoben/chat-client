package client

import (
	"fmt"
	"io"
	"net"
)

type chatService struct {
	net.Dialer
	endpoint       *endpoint
	conn           net.Conn
	connBufferSize int

	serviceChan chan []byte
	quit        *chan struct{}
}

type Option func(*chatService)

func NewChatService(options ...Option) (*chatService, error) {

	svc := chatService{
		serviceChan: make(chan []byte),
	}
	// Apply all
	for _, opt := range options {
		opt(&svc)
	}

	// Set default values if there still is not endpoint configured
	if svc.endpoint == nil {
		e := NewEndpoint(&EndpointOptions{})
		svc.endpoint = &e
	}

	return &svc, nil
}

/*
connect establishes a connection with endpoint and returns an error if unsuccessful.
if successful the connection is saved to the chatService.Connections instance
*/
func (c *chatService) connect(user string) error {
	conn, err := c.Dialer.Dial(c.endpoint.protocol, c.endpoint.String())
	if err != nil {
		return fmt.Errorf("could not connect to %s://%s: %s", c.endpoint.protocol, c.endpoint.String(), err)
	}

    helloMsg := newRawHelloMsg(user)
    fmt.Println(string(helloMsg))
    if _, err := conn.Write(helloMsg); err != nil {
        return fmt.Errorf("could not send hello message:", err)
    }

    buffer := make([]byte, c.connBufferSize) 

    // TODO: We might have to implement a timeout here
    n, err := conn.Read(buffer)
    if err != nil {
        return fmt.Errorf("could not read message from the server: %s", err)
    }

    ok, msgType := isHello(buffer[:n])
    if !ok {
        return fmt.Errorf("could not perform handshake, expected hello type message but received %s type", msgType)
    }

    fmt.Println("connected to", c.endpoint)
    c.conn = conn
	return nil
}

/*
sends a message over an established connection
*/
func (c *chatService) transmit(name string, body []byte) error {
    if name == "" {
        return fmt.Errorf("name cannot be empty")
    }

    if len(body) == 0 {
        return fmt.Errorf("rawString cannot be empty")
    }
    
    rawStr := newRawMsg(name, string(body))
    
    c.conn.Write(rawStr)

    return nil
}

/*
receive dispatches any incoming messages the service channel
*/
func (c *chatService) receive() {
	defer close(c.serviceChan)

	buffer := make([]byte, c.connBufferSize)
	for {
		select {
		case <-*c.quit:
			return
		default:
			n, err := c.conn.Read(buffer)
			if err != nil {
				if err == io.EOF {
					fmt.Printf("end of data stream reached")
				} else {
					fmt.Printf("could not read data: %v\n", err)
				}
				return
			}
			msg := buffer[:n]
			c.serviceChan <- msg
		}
	}
}

func (c *chatService) close() {
	c.conn.Close()
	close(c.serviceChan)
}

func WithEndpoint(endpoint *endpoint) Option {
	return func(cs *chatService) {
		cs.endpoint = endpoint
	}
}

func WithBufferSize(bufferSize int) Option {
	return func(cs *chatService) {
		cs.connBufferSize = bufferSize
	}
}
