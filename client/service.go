package client

// TOOD:
// - Fix bux where quit signalling does not properly work
// - Fix bug where inbound network connections are not written to stdout

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
    defaultConcurrency       = 1
    defaultConnBufferSize    = 1<<10
    defaultChannelBufferSize = 20
)

type chatService struct {
	net.Dialer
	endpoint       *endpoint
	conn           net.Conn
	connBufferSize int
    concurrency    int

	inboundCh   *messageBus
	outboundCh  *messageBus
	quit        chan struct{}
}

type Option func(*chatService)

func NewChatService(options ...Option) (*chatService, error) {

	svc := chatService{
        quit: make(chan struct{}, 1),
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

/*
start acts as an entry point to all of the go routine dispatching logic.
for reading data sources one go routine is created per source (stdin & c.conn) for asychonisity purposes.

workers that write to destinations are dispatched according to c.concurrency.

start also owns the responsibility of closing all channels once all go rountes have returned. This should only happen
when quit is signalled.
*/
func (c chatService) start(username string) {
    var wg sync.WaitGroup

    defer close(*c.inboundCh)
    defer log.Print("closing inboundChannel")

    defer close(*c.outboundCh)
    defer log.Print("closing outboundChannel")
    
    // Start ConnectionReader
    go func(){
	    for {
	    	select {
	    	case <-c.quit:
                log.Println("connection reader returning...")
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
                    }
                    continue
                }
                
                m, err := unmarshalMessage(chunk[:n])
                if err != nil {
                    log.Print(err)
                    continue
                }
                log.Printf("received %d bytes from server", n)
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
                case <- c.quit:
                log.Println("stdin reader returning...")
                    return
                default:
                    reader := bufio.NewReader(os.Stdin)
                    n, err := reader.Read(chunk)
                    if err != nil {
                        log.Printf("could not read from %v: %s", os.Stdin, err)
                    }
                    m := newMsg(username, strings.TrimSuffix(string(chunk[:n]), "\n"))
                    *c.outboundCh <- m
            }
        }
    }()

    // Reads from outbound channel and writes to net.Conn
    log.Print("Starting connectionWriter")
    connectionWriter := worker(writer) 
    c.dispatch(connectionWriter, username, c.outboundCh, c.conn, &wg)

    // Reads from inbound channel and writes to os.Stdout
    log.Print("Starting stdoutWriter")
    stdoutWriter := worker(writer) 
    c.dispatch(stdoutWriter, username, c.inboundCh, os.Stdout, &wg)

    log.Print("all workers spawned")
    wg.Wait()
}

type worker func(string, *messageBus, io.Writer, *sync.WaitGroup, *chan struct{})

func (c chatService)dispatch(w worker, name string, source *messageBus, dest io.Writer, wg *sync.WaitGroup) {
    for i:=0;i<c.concurrency;i++ {
        wg.Add(1)
        go w(name, source, dest, wg, &c.quit)
    }
}

func writer(name string, bus *messageBus, w io.Writer, wg *sync.WaitGroup, quit *chan struct{}) {
    defer wg.Done()

    for {
        select {
            case msg, ok := <-*bus:
                if !ok {
                    log.Print("worker could not read from closed channel")
                    log.Print("attempting to gracefully shutdown...")
                    *quit<-struct{}{}
                }
                rawMsg, err := msg.Bytes()
                if err != nil {
                    log.Printf("worker could not unmarshal recieved message: %s", err)
                    continue
                }
                 
                // TODO: Perhaps we should treat this as a transient error
                log.Printf("worker: writing message to %T", w)
                if _, err := w.Write(rawMsg); err != nil {
                    log.Printf("could not write to %T: %s", w, err)
                }
            case <- *quit:
                log.Print("worker returning...")
                return
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
