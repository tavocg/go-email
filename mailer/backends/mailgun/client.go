// Package mailgun sends mail through the Mailgun HTTP API.
package mailgun

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	mailgunsdk "github.com/mailgun/mailgun-go/v5"
	"github.com/tavocg/go-email/mailer"
)

// Client sends email through Mailgun.
type Client struct {
	domain string
	client mailgunsdk.Mailgun
}

// Option customizes a Client during construction.
type Option func(*Client) error

// WithAPIBase configures a custom Mailgun API base URL, such as the EU API
// endpoint.
func WithAPIBase(url string) Option {
	return func(c *Client) error {
		if strings.TrimSpace(url) == "" {
			return nil
		}
		return c.client.SetAPIBase(url)
	}
}

// WithHTTPClient configures the HTTP client used by the Mailgun SDK.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) error {
		if client != nil {
			c.client.SetHTTPClient(client)
		}
		return nil
	}
}

// NewClient creates a Client for a Mailgun sending domain.
func NewClient(domain, apiKey string, opts ...Option) (*Client, error) {
	client := &Client{
		domain: domain,
		client: mailgunsdk.NewMailgun(apiKey),
	}

	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, err
		}
	}

	return client, nil
}

// Send delivers the message using the Mailgun HTTP API.
func (c *Client) Send(ctx context.Context, message *mailer.Message) error {
	if ctx == nil {
		return mailer.ErrNilContext
	}
	if message == nil {
		return mailer.ErrNilMessage
	}

	data, err := message.Bytes()
	if err != nil {
		return err
	}

	mgMessage := mailgunsdk.NewMIMEMessage(
		c.domain,
		io.NopCloser(bytes.NewReader(data)),
		message.Recipients()...,
	)

	_, err = c.client.Send(ctx, mgMessage)
	return err
}
