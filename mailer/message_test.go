package mailer

import (
	"mime"
	"net/mail"
	"strings"
	"testing"
)

func TestMessageBytesRejectsHeaderNewlines(t *testing.T) {
	tests := []struct {
		name    string
		message *Message
	}{
		{
			name:    "from",
			message: NewPlainMessage("sender@example.com\r\nX-Injected: yes", []string{"recipient@example.com"}, "hello", "body"),
		},
		{
			name:    "to",
			message: NewPlainMessage("sender@example.com", []string{"recipient@example.com\r\nX-Injected: yes"}, "hello", "body"),
		},
		{
			name: "cc",
			message: &Message{
				From:      "sender@example.com",
				To:        []string{"recipient@example.com"},
				Cc:        []string{"copy@example.com\r\nX-Injected: yes"},
				Subject:   "hello",
				PlainBody: "body",
			},
		},
		{
			name: "bcc",
			message: &Message{
				From:      "sender@example.com",
				To:        []string{"recipient@example.com"},
				Bcc:       []string{"blind@example.com\r\nX-Injected: yes"},
				Subject:   "hello",
				PlainBody: "body",
			},
		},
		{
			name:    "subject",
			message: NewPlainMessage("sender@example.com", []string{"recipient@example.com"}, "hello\r\nX-Injected: yes", "body"),
		},
		{
			name: "attachment filename",
			message: func() *Message {
				message := NewPlainMessage("sender@example.com", []string{"recipient@example.com"}, "hello", "body")
				message.Attach("report.txt\r\nX-Injected: yes", []byte("contents"))
				return message
			}(),
		},
		{
			name: "attachment content type",
			message: func() *Message {
				message := NewPlainMessage("sender@example.com", []string{"recipient@example.com"}, "hello", "body")
				message.AttachWithType("report.txt", "text/plain\r\nX-Injected: yes", []byte("contents"))
				return message
			}(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := test.message.Bytes()
			if err != ErrInvalidHeader {
				t.Fatalf("Bytes() error = %v, want %v; rendered:\n%s", err, ErrInvalidHeader, data)
			}
		})
	}
}

func TestMessageBytesAllowsBodyNewlines(t *testing.T) {
	message := NewPlainMessage("sender@example.com", []string{"recipient@example.com"}, "hello", "line one\r\nline two")

	data, err := message.Bytes()
	if err != nil {
		t.Fatalf("Bytes() error = %v, want nil", err)
	}
	if !strings.Contains(string(data), "line one") || !strings.Contains(string(data), "line two") {
		t.Fatalf("Bytes() did not render expected body lines:\n%s", data)
	}
}

func TestMessageRecipientsIncludesBcc(t *testing.T) {
	message := &Message{
		To:  []string{"to@example.com"},
		Cc:  []string{"cc@example.com"},
		Bcc: []string{"bcc@example.com"},
	}

	got := message.Recipients()
	want := []string{"to@example.com", "cc@example.com", "bcc@example.com"}

	if len(got) != len(want) {
		t.Fatalf("Recipients() len = %d, want %d", len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("Recipients()[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestMessageBytesDoesNotRenderBccHeader(t *testing.T) {
	message := &Message{
		From:      "sender@example.com",
		To:        []string{"to@example.com"},
		Bcc:       []string{"bcc@example.com"},
		Subject:   "hello",
		PlainBody: "body",
	}

	data, err := message.Bytes()
	if err != nil {
		t.Fatalf("Bytes() error = %v, want nil", err)
	}
	parsed, err := mail.ReadMessage(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("ReadMessage() error = %v, want nil", err)
	}
	if got := parsed.Header.Get("Bcc"); got != "" {
		t.Fatalf("Bcc header = %q, want empty", got)
	}
}

func TestMessageBytesEncodesNonASCIISubject(t *testing.T) {
	message := NewPlainMessage("sender@example.com", []string{"recipient@example.com"}, "Olá", "body")

	data, err := message.Bytes()
	if err != nil {
		t.Fatalf("Bytes() error = %v, want nil", err)
	}
	parsed, err := mail.ReadMessage(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("ReadMessage() error = %v, want nil", err)
	}
	got, err := new(mime.WordDecoder).DecodeHeader(parsed.Header.Get("Subject"))
	if err != nil {
		t.Fatalf("DecodeHeader() error = %v, want nil", err)
	}
	if want := "Olá"; got != want {
		t.Fatalf("Subject header = %q, want %q", got, want)
	}
}

func TestMessageAttachInfersContentTypeFromFilename(t *testing.T) {
	message := NewPlainMessage("sender@example.com", []string{"recipient@example.com"}, "hello", "body")

	message.Attach("report.json", []byte{0x00, 0x01, 0x02, 0x03})

	if got, want := message.Attachments[0].ContentType, "application/json"; got != want {
		t.Fatalf("attachment content type = %q, want %q", got, want)
	}
}
