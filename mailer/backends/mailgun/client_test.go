package mailgun

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestSendRejectsHeaderInjectionBeforeRequest(t *testing.T) {
	client, err := NewClient("mg.example.com", "key", WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Error("invalid MIME message reached the Mailgun API")
			return nil, errors.New("unexpected HTTP request")
		}),
	}))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	message := mailer.NewPlainMessage("sender@example.com", []string{"recipient@example.com"}, "subject\r\nX-Injected: yes", "body")
	if err := client.Send(ctx, message); !errors.Is(err, mailer.ErrInvalidHeader) {
		t.Fatalf("Send() error = %v, want ErrInvalidHeader", err)
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
