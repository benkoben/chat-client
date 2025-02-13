package main

import (
	"chat-client/client"
	"chat-client/user"
	"fmt"
    "log"
)

const (
	host = "localhost"
	port = "7007"
)

func main() {
	user := user.User{}
	user.Login()

	if user.Name != "" {
		fmt.Println("Welcome", user)
	}

    endpoint := client.NewEndpoint(
        &client.EndpointOptions{
            Server: host,
            Port: port,
            Protocol: "tcp",
        },
    )

    svc, err := client.NewChatService(user.Name,
        client.WithEndpoint(&endpoint),
        client.WithBufferSize(1<<10),
        client.WithConcurrency(5),
    ) 
    if err != nil {
        log.Fatal(err)
    }

	c, err := client.NewClient(user.Name, svc)
	if err != nil {
		log.Fatal(err)
	}

	if err := c.Start(); err != nil {
		log.Fatal(err)
	}
}
