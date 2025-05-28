package main

import "fmt"

func main() {
	fmt.Println("Hello, Go!")
}

type Greeter struct {
	Greeting string
}

func (g Greeter) Greet() string {
	return "Hello, " + g.Greeting
}
