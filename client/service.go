package client

// TOOD:
// - Add timeouts to connections (server side), or perhaps a keepalive mechanism to determine wether or not a connection is active

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
    defaultConcurrency       = 2     // Amount of workers (inbound*n + outbound*n = min(2)) spawned
    defaultConnBufferSize    = 1<<10 // How large an incoming or outbound message can be
    defaultChannelBufferSize = 5     // The unhandled messages that can be buffered
)

type chatService struct {
	endpoint       *endpoint
    clientName     string
	conn           net.Conn
	connBufferSize int
    concurrency    int

	inboundCh   *messageBus
	outboundCh  *messageBus
	quit        *chan bool // Entrypoint from client to chatService SIGTERM or SIGINT signals are received
}

// func (c chatService)String()string {
//     return fmt.Sprintf("endpoint: %v, clientName %s, conn %v, connBufferSize %d, concurrency %d, inboundCh %v, outboundCh %v, quit %v", c.endpoint, c.clientName, c.conn, c.connBufferSize, c.concurrency, c.inboundCh, c.outboundCh, c.quit)
// }

type Option func(*chatService)

func NewChatService(clientName string, options ...Option) (*chatService, error) {
    quit := make(chan bool)
	svc := chatService{
        clientName: clientName,
        quit: &quit,
        concurrency: defaultConcurrency,
        connBufferSize: defaultConnBufferSize,
        inboundCh: newMessageBus(defaultChannelBufferSize),
        outboundCh: newMessageBus(defaultChannelBufferSize),
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
func (c *chatService) Connect(dialer net.Dialer) error {
	conn, err := dialer.Dial(c.endpoint.protocol, c.endpoint.String())
	if err != nil {
		return fmt.Errorf("could not connect to %s://%s: %s", c.endpoint.protocol, c.endpoint.String(), err)
	}

    helloMsg := newRawHelloMsg(c.clientName)
    if _, err := conn.Write(helloMsg); err != nil {
        return fmt.Errorf("could not send hello message: %s", err)
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

    c.conn = conn

	return nil
}

/*
start acts as an entry point to all of the go routine dispatching logic.
for reading data sources one go routine is created per source (stdin & c.conn) for asychonisity purposes.

workers that write to destinations are dispatched according to c.concurrency.

start also owns the responsibility of closing all channels once all go rountes have returned. This should only happen
when quit is signalled.
*/
func (c chatService) Start() {
    wg := sync.WaitGroup{}

    defer close(*c.inboundCh)
    defer close(*c.outboundCh)
    
    // Start ConnectionReader
    wg.Add(1)
    go func(){
        defer wg.Done()
	    for {
	    	select {
	    	case <-*c.quit:
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
                        *c.quit<-true
                    }
                    continue
                }
                
                m, err := unmarshalMessage(chunk[:n])
                if err != nil {
                    log.Print(err)
                    continue
                }
                *c.inboundCh <- m
	    	}
	    }
    }()

    // Start a StdinReader
    wg.Add(1)
    go func(){
        defer wg.Done()
        for {
            chunk := make([]byte, c.connBufferSize)
            select {
                case <-*c.quit:
                    return
                default:
                    reader := bufio.NewReader(os.Stdin)
                    n, err := reader.Read(chunk)
                    if err != nil {
                        log.Printf("could not read from %v: %s", os.Stdin, err)
                    }
                    m := newMsg(c.clientName, strings.TrimSuffix(string(chunk[:n]), "\n"))
                    *c.outboundCh <- m
            }
        }
    }()

    // Reads from outbound channel and writes to net.Conn
    for i:=0;i<c.concurrency;i++ {
        wg.Add(1)
        go worker(c.clientName, c.outboundCh, c.conn, &wg, c.quit)
    }

    // Reads from inbound channel and writes to os.Stdout
    for i:=0;i<c.concurrency;i++ {
        wg.Add(1)
        go worker(c.clientName, c.inboundCh, os.Stdout, &wg, c.quit)
    }

    fmt.Println("Connected!")
    fmt.Println()
    wg.Wait()
}

/*
Send a messageTypeBye message to the server, letting it know that the client will terminate the connection
*/
func (c chatService) Close() {
    close(*c.quit)
    // Terminate the connection and service instance
	c.conn.Close()
}

func worker(name string, bus *messageBus, w io.Writer, wg *sync.WaitGroup, quit *chan bool) {
    defer wg.Done()
    for {
        select {
            case msg, ok := <-*bus:
                if !ok {
                    log.Print("worker could not read from closed channel")
                    log.Print("attempting to gracefully shutdown...")
                    *quit<-true
                }

                var payload []byte
                
                switch w.(type) {
                    case *os.File:
                        payload = msg.BytesF()
                    case *net.TCPConn:
                        rawMsg, err := msg.Bytes()
                        if err != nil {
                            log.Printf("worker could not unmarshal recieved message: %s", err)
                            continue
                        }
                        payload = rawMsg
                }

                // TODO: Perhaps we should treat this as a transient error
                if _, err := w.Write(payload); err != nil {
                    log.Printf("could not write to %T: %s", w, err)
                }

            case <-*quit:
                return
        }
    }
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
