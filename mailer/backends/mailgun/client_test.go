package mailgun

import (
	"context"
	"testing"

	"github.com/tavocg/go-email/mailer"
)

func TestSendRejectsNilContext(t *testing.T) {
	client, err := NewClient("mg.example.com", "key")
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	err = client.Send(nil, mailer.NewPlainMessage("sender@example.com", []string{"recipient@example.com"}, "hello", "body"))

	if err != mailer.ErrNilContext {
		t.Fatalf("Send() error = %v, want %v", err, mailer.ErrNilContext)
	}
}

func TestSendRejectsNilMessage(t *testing.T) {
	client, err := NewClient("mg.example.com", "key")
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	err = client.Send(context.Background(), nil)

	if err != mailer.ErrNilMessage {
		t.Fatalf("Send() error = %v, want %v", err, mailer.ErrNilMessage)
	}
}
