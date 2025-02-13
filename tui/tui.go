package tui


// Trying out some currying logic
func log(formatter func(string, string) string) func(string, string){
    return func(first, second string) {
        fmt.Println(formatter(first, second)
    }
}

func commaDelimiter(first, second string) string {
    return first + "," + second
}

func colonDelimter(first, second string) string {
    return first + ":" + second
}
