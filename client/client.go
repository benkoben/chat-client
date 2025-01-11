package client

import (
	"fmt" 
    "os" 
    "os/signal"
	"sync"
	"syscall"
    "encoding/json"
)

/*
   Client primarily does two things 1. Reads user input, adds metadata to them and sends it to the server
   2. Reads service events and presents that to the user
*/

const (
	defaultServer = "localhost"
	defaultPort   = "7007"
)

var (
	globalQuit = make(chan struct{})
)


type client struct {
	name string 
    svc  *chatService
	quit *chan struct{}
}

type ClientSvcOptions struct {
	MessageSize int
}

func NewClient(username, server, port string) (*client, error) {

	quitChan := make(chan struct{}, 1)

	// Initialize service
	endpointOpts := &EndpointOptions{Server: server, Port: port}
	endpoint := NewEndpoint(endpointOpts)
	service, err := NewChatService(
		WithEndpoint(&endpoint),
		WithBufferSize(1<<10),
	)
	service.quit = &quitChan

	if err != nil {
		return nil, fmt.Errorf("could not initialize service: %s", err)
	}

	return &client{
		name: username,
		svc:  service,
		quit: &quitChan,
	}, nil
}

func (c client) Start() error {
	// Start a go routine that listens on the connection
	// and writes the received data to the serviceChannel
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	defer close(quit)

	fmt.Println("Connecting to", c.svc.endpoint)
	if err := c.svc.connect(); err != nil {
		return err
	}

	fmt.Println("Listening for incoming transmissions")
	go c.svc.receive()

	for {
		select {
		case rawMsg := <-c.svc.serviceChan:
			wg.Add(1)
			go func(data []byte) {
				defer wg.Done()
                var msg Message
                err := json.Unmarshal(data, &msg) 
                if err != nil {
                    fmt.Println("could not unmarshal incoming message:", err)
                }
				fmt.Println(msg)
			}(rawMsg)

		case sig := <-quit:
			fmt.Println()
			fmt.Println(sig)
			c.stop()
			return nil
		}
	}
}

func (c client) stop() {
	fmt.Println("Stopping chatService...")
	*c.quit <- struct{}{}
	return
}
