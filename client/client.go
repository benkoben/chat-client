package client

import (
	"fmt" 
    "os" 
    "os/signal"
	"syscall"
    "log"
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

	log.Print("Connecting to", c.svc.endpoint)
	if err := c.svc.connect(c.name); err != nil {
		return err
	}

	go c.svc.receive()
    go c.svc.outboundDispatcher(c.name)

    // I think it makes more sense to move this part into the service
    // because we are reading from the service channel.
    // 
    // It does not feel that having it here provides more value.
    // What do do instead:
    // - Launch two go routines, transmit and receive.
    // - Wait use wait groups to wait and if a quit signal comes in we close
    //   signal these go routines to cleanup
	for {
		select {
		case rawMsg := <-c.svc.inboundCh:
            msg, err := unmarshalMessage(rawMsg)

            if err != nil {
                fmt.Printf("could not read message: %s\n")
                continue
            }
            // Quick and dirty stdout
            // Here we eventually will add more fancy rendering stuff
            fmt.Println(msg)

		case sig := <-quit:
			fmt.Println()
			fmt.Println(sig)

            // close all service go routines and connection
            c.svc.close(c.name)

            // Stop chat service
			c.stop()
			return nil
		}
	}

    return nil
}

func (c client) stop() {
	fmt.Println("Stopping chatService...")
	return
}
