package client

import (
	"testing"
)

func TestNewChatService(t *testing.T) {
	cases := []struct {
		name    string
		options []Option
		want    *chatService
		wantErr error
	}{
		{
			name:    "NewChatService with default options",
			options: []Option{},
			want: &chatService{
				endpoint: &endpoint{"localhost", "7007", "tcp"},
			},
			wantErr: nil,
		},
		{
			name: "NewChatService with explicit endpoint",
			options: []Option{
				WithEndpoint(&endpoint{"127.0.0.1", "8888", "tcp"}),
			},
			want:    &chatService{endpoint: &endpoint{"127.0.0.1", "8888", "tcp"}},
			wantErr: nil,
		},
	}

	for _, tt := range cases {
		got, gotErr := NewChatService(tt.options...)
		if *got.endpoint != *tt.want.endpoint {
			t.Errorf("%s -> NewChatService(%v): got %v, want %v", tt.name, tt.options, got, tt.want)
		}

		if gotErr == nil && tt.wantErr != nil {
			t.Errorf("%s -> NewChatService(%v): expected and error but none was received", tt.name, tt.options)
		}
	}
}
