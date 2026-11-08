package smtp

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"net/textproto"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/tavocg/go-email/mailer"
)

func TestSendUsesMailboxEnvelope(t *testing.T) {
	certificateServer := httptest.NewTLSServer(http.NotFoundHandler())
	serverTLS := certificateServer.TLS.Clone()
	roots := x509.NewCertPool()
	roots.AddCert(certificateServer.Certificate())
	certificateServer.Close()

	for _, transport := range []string{"implicit TLS", "STARTTLS"} {
		t.Run(transport, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { listener.Close() })
			type result struct {
				data []byte
				err  error
			}
			done := make(chan result, 1)
			startTLS := transport == "STARTTLS"
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					done <- result{err: err}
					return
				}
				data, err := serveEnvelope(conn, serverTLS, startTLS)
				done <- result{data, err}
			}()
			opts := []Option{WithTLSConfig(&tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12})}
			if startTLS {
				opts = append(opts, WithStartTLS())
			}
			client, err := NewClient(listener.Addr().String(), "", "", opts...)
			if err != nil {
				t.Fatal(err)
			}
			message := mailer.NewPlainMessage(`"Sender, Example" <sender@example.com>`, []string{"Recipient <recipient@example.com>"}, "hello", "body")
			message.Cc = []string{`Copy <"copy name"@example.com>`}
			message.Bcc = []string{"=?UTF-8?Q?Jos=C3=A9?= <hidden@example.com>"}
			from, recipients := message.From, message.Recipients()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := client.Send(ctx, message); err != nil {
				t.Errorf("Send() error = %v", err)
			}
			var got result
			select {
			case got = <-done:
			case <-ctx.Done():
				t.Fatal("SMTP fixture did not finish before the deadline")
			}
			if got.err != nil {
				t.Fatalf("SMTP fixture: %v", got.err)
			}
			parsed, err := mail.ReadMessage(bytes.NewReader(got.data))
			if err != nil {
				t.Fatal(err)
			}
			for key, want := range map[string]string{"From": from, "To": message.To[0], "Cc": message.Cc[0]} {
				if got := parsed.Header.Get(key); got != want {
					t.Errorf("%s header = %q, want %q", key, got, want)
				}
			}
			if strings.Contains(string(got.data), "hidden@example.com") || parsed.Header.Get("Bcc") != "" {
				t.Error("Bcc recipient leaked into MIME data")
			}
			if message.From != from || !reflect.DeepEqual(message.Recipients(), recipients) {
				t.Error("Send mutated the message's addresses")
			}
		})
	}
}

func TestEnvelopeAddress(t *testing.T) {
	for _, tt := range []struct{ input, want string }{
		{"user@example.com", "user@example.com"},
		{" User.Name+tag@Example.com ", "User.Name+tag@Example.com"},
		{"Name <user@example.com>", "user@example.com"},
		{`"Last, First" <user@example.com>`, "user@example.com"},
		{"=?UTF-8?Q?Jos=C3=A9?= <user@example.com>", "user@example.com"},
		{`Name <"local part"@example.com>`, `"local part"@example.com`},
		{`Name <"local@part"@example.com>`, `"local@part"@example.com`},
	} {
		t.Run(tt.input, func(t *testing.T) {
			if got, err := envelopeAddress(tt.input); err != nil || got != tt.want {
				t.Fatalf("envelopeAddress() = %q, %v, want %q, nil", got, err, tt.want)
			}
		})
	}
}

func TestSendRejectsInvalidEnvelopeBeforeDial(t *testing.T) {
	for _, field := range []string{"From", "To", "Cc", "Bcc"} {
		for _, invalid := range []string{"not-an-address", "a@example.com, b@example.com"} {
			t.Run(field+"/"+invalid, func(t *testing.T) {
				message := mailer.NewPlainMessage("sender@example.com", []string{"recipient@example.com"}, "hello", "body")
				switch field {
				case "From":
					message.From = invalid
				case "To":
					message.To = []string{invalid}
				case "Cc":
					message.Cc = []string{invalid}
				case "Bcc":
					message.Bcc = []string{invalid}
				}
				// A dial would fail: the zero-value client has no server address.
				client := &Client{}
				if err := client.Send(context.Background(), message); !errors.Is(err, mailer.ErrInvalidEnvelopeAddress) {
					t.Fatalf("Send() error = %v, want ErrInvalidEnvelopeAddress", err)
				}
			})
		}
	}
}

func TestSendRejectsEnvelopeHeaderInjection(t *testing.T) {
	message := mailer.NewPlainMessage("sender@example.com", []string{"recipient@example.com\r\nRCPT TO:<hidden@example.com>"}, "hello", "body")
	client := &Client{}
	if err := client.Send(context.Background(), message); !errors.Is(err, mailer.ErrInvalidHeader) {
		t.Fatalf("Send() error = %v, want ErrInvalidHeader", err)
	}
}

func serveEnvelope(conn net.Conn, config *tls.Config, startTLS bool) ([]byte, error) {
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, err
	}
	if !startTLS {
		conn = tls.Server(conn, config)
	}
	wire := textproto.NewConn(conn)
	command := func(want, response string) error {
		got, err := wire.ReadLine()
		if err != nil {
			return err
		}
		if got != want {
			return fmt.Errorf("command = %q, want %q", got, want)
		}
		return wire.PrintfLine("%s", response)
	}
	if err := wire.PrintfLine("220 localhost ready"); err != nil {
		return nil, err
	}
	if startTLS {
		if err := command("EHLO localhost", "250-localhost\r\n250 STARTTLS"); err != nil {
			return nil, err
		}
		if err := command("STARTTLS", "220 ready for TLS"); err != nil {
			return nil, err
		}
		wire = textproto.NewConn(tls.Server(conn, config))
	}
	for _, step := range []struct{ request, response string }{
		{"EHLO localhost", "250 localhost"},
		{"MAIL FROM:<sender@example.com>", "250 sender accepted"},
		{"RCPT TO:<recipient@example.com>", "250 recipient accepted"},
		{`RCPT TO:<"copy name"@example.com>`, "250 recipient accepted"},
		{"RCPT TO:<hidden@example.com>", "250 recipient accepted"},
		{"DATA", "354 send message"},
	} {
		if err := command(step.request, step.response); err != nil {
			return nil, err
		}
	}
	data, err := wire.ReadDotBytes()
	if err != nil {
		return nil, err
	}
	if err := wire.PrintfLine("250 queued"); err != nil {
		return nil, err
	}
	return data, command("QUIT", "221 bye")
}
