package client

import (
	"testing"
    "net"

    "github.com/google/go-cmp/cmp"
    "github.com/google/go-cmp/cmp/cmpopts"
)

var (
    mockInboundCh = newMessageBus(1)
    mockOutboundCh = newMessageBus(1)
    mockQuitCh = make(chan bool)
)

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

func TestConnect(t *testing.T) {
    cases := []struct{
        name string
        chatService chatService
        host string
        port string
        want string
    }{
        {
            name: "connect to server",
			chatService: chatService{
                clientName: "benko",
                quit: &mockQuitCh,
                concurrency: 2,
                connBufferSize: 1<<8,
                conn: nil,
                inboundCh: mockInboundCh,
                outboundCh: mockOutboundCh,
                endpoint: &endpoint{"localhost", "7007", "tcp"},
            },
            
        }
    }


}
