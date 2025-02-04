package client

import (
    "strings"
	"fmt"
	"net"
    "bufio"
    "log"
    "io"
    "os"
    "sync"
)

const (
    defaultConcurrency = 1
    defaultConnBufferSize = 1<<10
)

type chatService struct {
	net.Dialer
	endpoint       *endpoint
	conn           net.Conn
	connBufferSize int
    concurrency int

	inboundCh chan []byte
	quit        chan struct{}
}

type Option func(*chatService)

func NewChatService(options ...Option) (*chatService, error) {

	svc := chatService{
		inboundCh: make(chan []byte, 0), // Capacity is dynamically adjusted
        quit: make(chan struct{}, 1),
        concurrency: defaultConcurrency,
        connBufferSize: defaultConnBufferSize,
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

    log.Print("Connected!")
    c.conn = conn
	return nil
}

// Implement fan-out pattern
// Whenever a message is received on the outboundCh then create a transmitter worker

func (c chatService) outboundDispatcher(username string){
    outCh := make(chan []byte, 100)
    var wg sync.WaitGroup
    
    wg.Add(1)
    // Read Stdin 
    go func(){
        defer wg.Done()
        for {
            chunk := make([]byte, c.connBufferSize)
            select {
                case <- c.quit:
                    return
                default:
                    reader := bufio.NewReader(os.Stdin)
                    n, err := reader.Read(chunk)
                    if err != nil {
                        log.Printf("could not read from %v: %s", os.Stdin, err)
                    }

                    // TODO: There is probably a better way to do this
                    rawMsg := chunk[:n]
                    outCh <- []byte(strings.TrimSuffix(string(rawMsg), "\n"))
                    continue
            }
        }
    }()

    for i:=0;i<c.concurrency;i++ {
        wg.Add(1)
        go transmit(username, outCh, c.conn, &wg, &c.quit)
    } 

    wg.Wait()
}

/*
transmit is the consumer of the outboundCh. It transforms inbound []byte into a message before writing that to c.conn
if any error occurs during this time it will be logged to stdout.
*/
func transmit(name string, outCh <-chan []byte, conn net.Conn, wg *sync.WaitGroup, quit *chan struct{}) {
    // Note to self: I want to try out having pointer to quit and wg. 
    // If that does not work then we can just do a anonymous go routine like I did for reading stdin
    
    defer wg.Done()
    for {
        select {
            case <- *quit:
                return
            case body := <-outCh:
                msg := newRawMsg(name, string(body))

                _, err := conn.Write(msg)
                if err != nil {
                    log.Printf("could not write to connection: %s", err)
                }
        }
    }
}

/*
receive dispatches any incoming messages the service channel
*/
func (c *chatService) receive() {
	for {
		select {
		case <-c.quit:
			return
		default:
            // Read the data from the packet
            reader := bufio.NewReader(c.conn)
            chunk := make([]byte, c.connBufferSize)
            n, err := reader.Read(chunk)

            // Whenever an error happens 
            if err != nil {

                // If the server closes the connection
                if err == io.EOF {
                    log.Print("Server closed connection")
                    c.quit<-struct{}{}
                    // TODO: Initiate cleanups
                }
                continue
            }

            // Add the data to the message buffer
            msg := chunk[:n]
            c.inboundCh <- msg
		}
	}
}

/*
Send a messageTypeBye message to the server, letting it know that the client will terminate the connection
*/
func (c *chatService) close(user string) {
    c.quit <- struct{}{}
    // Send a bye type message to the server
    byeMsg, err := newRawSystemMsg(user, msgTypeBye)
    if err != nil {
        log.Print(err)
    }
    log.Print("Sending bye to server")
    c.conn.Write(byeMsg)

    log.Print("Closing connection...")
    // Terminate the connection and service instance
	c.conn.Close()
	close(c.inboundCh)
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

func WithConcurrency(concurrency int) Option {
	return func(cs *chatService) {
		cs.concurrency = concurrency
	}
}
