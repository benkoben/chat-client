package client

import (
	"fmt" 
    "os" 
    "os/signal"
	"syscall"
)

/*
   Client primarily does two things 1. Reads user input, adds metadata to them and sends it to the server
   2. Reads service events and presents that to the user
*/

const (
	defaultServer = "localhost"
	defaultPort   = "7007"
)

type client struct {
	name string 
    svc  *chatService
}

type ClientSvcOptions struct {
	MessageSize int
}

func NewClient(username, server, port string) (*client, error) {

	// Initialize service
	endpointOpts := &EndpointOptions{Server: server, Port: port}
	endpoint := NewEndpoint(endpointOpts)
	service, err := NewChatService(
		WithEndpoint(&endpoint),
		WithBufferSize(1<<10),
	)

	if err != nil {
		return nil, fmt.Errorf("could not initialize service: %s", err)
	}

	return &client{
		name: username,
		svc:  service,
	}, nil
}

func (c client) Start() error {
	// Start a go routine that listens on the connection
	// and writes the received data to the serviceChannel
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	defer close(quit)

	fmt.Println("Connecting to", c.svc.endpoint)
	if err := c.svc.connect(c.name); err != nil {
		return err
	}
    
    // Watch for SIGNINT and SIGTERM signals
    go func(){
        for {
            select {
                case <-quit:
                    c.svc.close(c.name)
                    return
            }
        }
    }()
    
    // start service
    c.svc.start(c.name)

    fmt.Println("Goodbye", c.name)

    return nil
}
