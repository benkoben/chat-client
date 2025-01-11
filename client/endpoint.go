package client

import (
	"fmt"
)

// There should be a dedicated go routine that writes to the screen
// There should be another go routine that reads inbound messages
const (
	defaultServerEndpoint   string = "localhost"
	defaultPortEndpoint     string = "7007"
	defaultProtocolEndpoint string = "tcp"
)

type endpoint struct {
	server   string
	port     string
	protocol string
}

type EndpointOptions struct {
	Server   string
	Port     string
	Protocol string
}

func (e endpoint) String() string {
	return fmt.Sprintf("%s:%s", e.server, e.port)
}

func NewEndpoint(opts *EndpointOptions) endpoint {
	if opts.Server == "" {
		opts.Server = defaultServerEndpoint
	}
	if opts.Port == "" {
		opts.Port = defaultPortEndpoint
	}
	if opts.Protocol == "" {
		opts.Protocol = defaultProtocolEndpoint
	}

	return endpoint{opts.Server, opts.Port, opts.Protocol}
}
