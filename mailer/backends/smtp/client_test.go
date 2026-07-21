package smtp

import (
	"context"
	"testing"

	"github.com/tavocg/go-email/mailer"
)

func TestNewClientRejectsInvalidAddress(t *testing.T) {
	client, err := NewClient("smtp.example.com", "", "")
	if err != mailer.ErrInvalidAddress {
		t.Fatalf("NewClient() error = %v, want %v", err, mailer.ErrInvalidAddress)
	}
	if client != nil {
		t.Fatalf("NewClient() client = %#v, want nil", client)
	}
}

func TestNewClientDoesNotProbeTransport(t *testing.T) {
	address := "127.0.0.1:1"

	client, err := NewClient(address, "user", "password", WithStartTLS())
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}
	if client == nil {
		t.Fatal("NewClient() client = nil, want client")
	}
	if got := client.Address(); got != address {
		t.Fatalf("Address() = %q, want %q", got, address)
	}
	if got := client.User(); got != "user" {
		t.Fatalf("User() = %q, want %q", got, "user")
	}
	if !client.StartTLS() {
		t.Fatal("StartTLS() = false, want true")
	}
}

func TestCheckTransportRejectsNilContext(t *testing.T) {
	client, err := NewClient("127.0.0.1:465", "", "")
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	err = client.CheckTransport(nil)

	if err != mailer.ErrNilContext {
		t.Fatalf("CheckTransport() error = %v, want %v", err, mailer.ErrNilContext)
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
