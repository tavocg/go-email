package main

import (
	"context"
	"log"
	"time"

	"github.com/tavocg/go-email/mailer"
	smtpbackend "github.com/tavocg/go-email/mailer/backends/smtp"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := smtpbackend.NewClient(
		"smtp.example.com:587",
		"username",
		"password",
		smtpbackend.WithStartTLS(),
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := client.CheckTransport(ctx); err != nil {
		log.Fatal(err)
	}

	message := mailer.NewPlainMessage(
		"sender@example.com",
		[]string{"recipient@example.com"},
		"Hello",
		"Hello from go-email.",
	)

	if err := client.Send(ctx, message); err != nil {
		log.Fatal(err)
	}
}
