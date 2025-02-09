package client

import (
	"fmt" 
    "os" 
    "os/signal"
	"syscall"
    "errors"
    "net"
)

type ChatService interface {
    Connect(net.Dialer) error
    Start()
    Close()
}

type client struct {
	name string 
    svc  ChatService
}

func NewClient(username string, service ChatService) (*client, error) {

    if service == nil {
        return nil, errors.New("service cannot be nil")
    }

    if username == "" {
        return nil, errors.New("username cannot be nil")
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
    dialer := net.Dialer{}

	defer close(quit)

	if err := c.svc.Connect(dialer); err != nil {
		return err
	}
    
    // Watch for SIGNINT and SIGTERM signals
    go func(){
        for {
            select {
                case <-quit:
                    c.svc.Close()
                    return
            }
        }
    }()
    
    // start service
    c.svc.Start()

    fmt.Println("Goodbye", c.name)

    return nil
}
