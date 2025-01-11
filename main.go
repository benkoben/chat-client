package main

import (
	"chat-client/client"
	"chat-client/user"
	"fmt"
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

	c, err := client.NewClient(user.Name, "localhost", "7007")
	if err != nil {
		panic(err)
	}

	if err := c.Start(); err != nil {
		panic(err)
	}
}
