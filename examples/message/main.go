package main

import (
	"fmt"
	"log"

	"github.com/tavocg/go-email/mailer"
)

func main() {
	message := mailer.NewAlternativeMessage(
		"sender@example.com",
		[]string{"recipient@example.com"},
		"Welcome",
		"Welcome to the service.",
		"<p>Welcome to the service.</p>",
	)
	message.AttachWithType("hello.txt", "text/plain", []byte("hello\n"))

	data, err := message.Bytes()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%d bytes\n", len(data))
}
