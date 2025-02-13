package client

import (
	"testing"
    "net"
    "io"
    "fmt"
    "errors"

    "github.com/google/go-cmp/cmp"
    "github.com/google/go-cmp/cmp/cmpopts"
)

var (
    mockInboundCh = newMessageBus(1)
    mockOutboundCh = newMessageBus(1)
    mockQuitCh = make(chan bool)
)

// Implements the Dialer interface
type mockClientServer struct{
    server net.Conn
    client net.Conn
}

func newMockClientServer() *mockClientServer {
    server, client := net.Pipe()
    return &mockClientServer{
        server: server,
        client: client,
    }
}

func mockServerHandshake(conn net.Conn, returnWrongMessage bool, returnEOF bool) error {

    var response []byte

    switch {
        case returnWrongMessage:
            response = newRawMsg("testing", "this message is wrong")
        default:
            response = newRawHelloMsg("system")
    }

	// Here we need to implement a handshake to exchange client information.
	helloMsgBuf := make([]byte, 1<<10)
    n, err := conn.Read(helloMsgBuf)
    if err != nil {
    	return fmt.Errorf("could not read the hello message from the client: %s", err)
    }
    
    ok, clientMsg := isHello(helloMsgBuf[:n])
    if !ok{
        return fmt.Errorf("could not perform handshake, expected message type hello but recieved %s", clientMsg)
    }

    if returnEOF {
        // Raise an error at the client side by closing the connection
        conn.Close()
        return nil
    }
    if _, err := io.WriteString(conn, string(response)); err != nil {
        return fmt.Errorf("could not respond to client: %s", err)
    }


    return nil
}

func TestNewChatService(t *testing.T) {
	cases := []struct {
		name    string
		options []Option
        clientName string
		want    *chatService
		wantErr error
	}{
		{
			name: "NewChatService with explicit endpoint",
            clientName: "benko",
			options: []Option{
                WithBufferSize(1<<8),
			},
			want:    &chatService{
                clientName: "benko",
                quit: &mockQuitCh,
                concurrency: 2,
                connBufferSize: 1<<8,
                conn: nil,
                inboundCh: mockInboundCh,
                outboundCh: mockOutboundCh,
                endpoint: &endpoint{"localhost", "7007", "tcp"},
            },
			wantErr: nil,
		},
	}

	for _, tt := range cases {
		got, gotErr := NewChatService(tt.clientName, tt.options...)

        if !cmp.Equal(got, tt.want, cmpopts.IgnoreUnexported(chatService{})) {
			t.Errorf("%s -> NewChatService(%v, %v): got %v, want %v", tt.name, tt.clientName, tt.options, got, tt.want)
        }

		if gotErr == nil && tt.wantErr != nil {
			t.Errorf("%s -> NewChatService(%v): expected and error but none was received", tt.name, tt.options)
		}
	}
}

func TestHandhake(t *testing.T) {
    cases := []struct{
        name string
        clientAndServer *mockClientServer
        returnEOF bool
        returnWrongMessage bool
        wantErr error
    }{
        {
            name: "Handshake with no error",
            clientAndServer: newMockClientServer(),
            returnEOF: false,
            returnWrongMessage: false,
            wantErr: nil,
        },
        {
            name: "Server closes to early",
            clientAndServer: newMockClientServer(),
            returnEOF: true,
            returnWrongMessage: false,
            wantErr: errors.New("could not read message from the server"),
        },
        {
            name: "Server sends back wrong message",
            clientAndServer: newMockClientServer(),
            returnEOF: false,
            returnWrongMessage: true,
            wantErr: errors.New("expected hello type message but received message type"),
        },
    }

    for _, tt := range cases {
        stop := make(chan bool)
        // start a new server
        go func(){
            for {
                select {
                    case<-stop:
                        return
                    default:
                        if err := mockServerHandshake(tt.clientAndServer.server, tt.returnWrongMessage, tt.returnEOF); err != nil {
                            t.Log(err)
                            continue
                        }

                } 
            }
        }()

        gotErr := handshake("test", tt.clientAndServer.client)

        if gotErr == nil && tt.wantErr != nil {
            t.Errorf("%s -> handshake(%s, %v) expected an error but none was returned", tt.name, "test", tt.clientAndServer.client)
        }
        close(stop)
    }
}
