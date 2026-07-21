package mailer

import (
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
