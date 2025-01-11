package client

import (
	"testing"
)

type fakeServiceClient struct{}

func (f *fakeServiceClient) connect() error {
	return nil
}

func (f *fakeServiceClient) transmit() error {
	return nil
}

func (f *fakeServiceClient) receive() {}
func (f *fakeServiceClient) quit()    {}

func TestNewClient(t *testing.T) {
	user := "benko"
	inputServiceClient := fakeServiceClient{}
	want := &client{name: user, svc: &inputServiceClient}

	got, _ := NewClient(user, &inputServiceClient)
	if *got != *want {
		t.Errorf("new chat client -> NewClient(%v, %v), got %v, want %v", user, inputServiceClient, got, want)
	}
}
