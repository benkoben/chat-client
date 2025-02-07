package user

import (
    "os"
    "bufio"
    "fmt"
    "strings"
    "unicode/utf8"
)

const (
	maxUsernameLength = 20
)

type User struct {
	Name string
}

func (u User) String() string {
	return u.Name
}

// Asks for a name
func (u *User) Login() {

    for {
    	reader := bufio.NewReader(os.Stdin)
    	fmt.Print("Username: ")
    	name, err := reader.ReadString('\n')

	    if err != nil {
	    	fmt.Println("Invalid username. Please try again.")
	    	fmt.Fprint(os.Stderr, err)
	    	continue
	    }

	    if strings.Split(name, " ")[0] != name {
	    	fmt.Println("Invalid username. Name cannot include a whitepace. Please try again.")
	    	continue
	    }

	    if utf8.RuneCountInString(name) > maxUsernameLength { // Account for string terminator
	    	fmt.Println("Invalid username. Name cannot be longer than 20 characters. Please try again.")
	    	continue
	    }
        
	    u.Name = strings.Split(name, "\n")[0]
        return
    }
}
