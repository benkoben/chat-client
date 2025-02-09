package client

import (
	"testing"
    "errors"
    "reflect"
    "net"
)


type MockService struct{}

func (m *MockService)Connect(net.Dialer) error {
    return nil
}
func (m MockService)Start() {}
func (m MockService)Close() {}

var (
    testQuitChan = make(chan bool)
    testMockService = MockService{}
)

func TestNewClient(t *testing.T) {

    tests := []struct{
        name string
        username string
        service ChatService
        want *client
        wantErr error
    }{
        {
            name: "new client with default values",
            username: "benko",
            service: &testMockService,
            want: &client{name: "benko", svc: &testMockService},
            wantErr: nil,
        },
        {
            name: "missing service",
            username: "benko",
            service: nil,
            want: nil,
            wantErr: errors.New("service cannot be nil"),
        },
        {
            name: "missing username",
            username: "",
            service: &testMockService,
            want: nil,
            wantErr: errors.New("username cannot be nil"),
        },
    }
    
    for _, tt := range tests {
        got, gotErr := NewClient(tt.username, tt.service)

        if !reflect.DeepEqual(got, tt.want) {
           t.Errorf("NewClient(%v, %v), got %v, want %v", tt.username, tt.service, got, tt.want) 
        }

        if gotErr == nil && tt.wantErr != nil {
           t.Errorf("NewClient(%v, %v) expected an error but none was received", tt.username, tt.service) 
        }
    }
}
