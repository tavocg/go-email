package smtp

import (
	"context"
	"testing"

	"github.com/tavocg/go-email/mailer"
)

func TestNewClientRejectsNilContext(t *testing.T) {
	client, err := NewClient(nil, "smtp.example.com:465", "", "")
	if err != mailer.ErrNilContext {
		t.Fatalf("NewClient() error = %v, want %v", err, mailer.ErrNilContext)
	}
	if client != nil {
		t.Fatalf("NewClient() client = %#v, want nil", client)
	}
}

func TestNewClientRejectsInvalidAddress(t *testing.T) {
	client, err := NewClient(context.Background(), "smtp.example.com", "", "")
	if err != mailer.ErrInvalidAddress {
		t.Fatalf("NewClient() error = %v, want %v", err, mailer.ErrInvalidAddress)
	}
	if client != nil {
		t.Fatalf("NewClient() client = %#v, want nil", client)
	}
}

func TestSendRejectsNilContext(t *testing.T) {
	client := &Client{}

	err := client.Send(nil, mailer.NewPlainMessage("sender@example.com", []string{"recipient@example.com"}, "hello", "body"))

	if err != mailer.ErrNilContext {
		t.Fatalf("Send() error = %v, want %v", err, mailer.ErrNilContext)
	}
}

func TestSendRejectsNilMessage(t *testing.T) {
	client := &Client{}

	err := client.Send(context.Background(), nil)

	if err != mailer.ErrNilMessage {
		t.Fatalf("Send() error = %v, want %v", err, mailer.ErrNilMessage)
	}
}
