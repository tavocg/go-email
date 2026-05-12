// Package smtp crafts messages and sends email
package smtp

import (
	"context"
	"crypto/tls"
	"net"
	"net/smtp"
	"time"
)

const (
	// ExtensionStartTLS is advertised by servers that support upgrading a plain
	// SMTP connection to TLS.
	ExtensionStartTLS = "STARTTLS"
)

// Client sends email over either implicit TLS or STARTTLS.
type Client struct {
	Address  string
	User     string
	Password string

	timeout    time.Duration
	tlsConfig  *tls.Config
	startTLS   bool
	serverName string
}

// Option customizes a Client during construction.
type Option func(*Client)

// WithTimeout sets the dial and probe timeout used by the client.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if timeout > 0 {
			c.timeout = timeout
		}
	}
}

// WithTLSConfig sets the TLS configuration used for implicit TLS and STARTTLS.
func WithTLSConfig(config *tls.Config) Option {
	return func(c *Client) {
		if config != nil {
			c.tlsConfig = config.Clone()
		}
	}
}

// WithStartTLS configures the client to connect in plaintext and upgrade with
// STARTTLS before sending.
func WithStartTLS() Option {
	return func(c *Client) {
		c.startTLS = true
	}
}

// NewClient creates a Client and verifies the configured encrypted transport.
func NewClient(ctx context.Context, address, user, password string, opts ...Option) (*Client, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, ErrInvalidAddress
	}

	client := &Client{
		Address:    address,
		User:       user,
		Password:   password,
		timeout:    10 * time.Second,
		serverName: host,
	}

	for _, opt := range opts {
		opt(client)
	}

	if client.tlsConfig == nil {
		client.tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: host,
		}
	} else if client.tlsConfig.ServerName == "" {
		client.tlsConfig = client.tlsConfig.Clone()
		client.tlsConfig.ServerName = host
	}

	if err := client.checkTransport(ctx); err != nil {
		return nil, err
	}

	return client, nil
}

// Send delivers the message using the configured encrypted transport.
func (c *Client) Send(ctx context.Context, message *Message) error {
	if ctx == nil {
		return ErrNilContext
	}
	if message == nil {
		return ErrNilMessage
	}

	data, err := message.bytes()
	if err != nil {
		return err
	}

	recipients := message.recipients()
	if len(recipients) == 0 {
		return ErrMissingRecipients
	}

	if c.startTLS {
		return c.sendStartTLS(ctx, message.From, recipients, data)
	}
	return c.sendImplicitTLS(ctx, message.From, recipients, data)
}

func (c *Client) checkTransport(ctx context.Context) error {
	if ctx == nil {
		return ErrNilContext
	}

	if c.startTLS {
		return c.checkStartTLS(ctx)
	}
	return c.checkImplicitTLS(ctx)
}

func (c *Client) checkStartTLS(ctx context.Context) error {
	conn, stop, err := c.dialPlain(ctx)
	if err != nil {
		return err
	}
	defer stop()
	defer conn.Close()

	client, err := smtp.NewClient(conn, c.serverName)
	if err != nil {
		return err
	}
	defer client.Close()

	if ok, _ := client.Extension(ExtensionStartTLS); !ok {
		return ErrMissingStartTLS
	}
	if err := client.StartTLS(c.tlsConfig.Clone()); err != nil {
		return err
	}
	return client.Quit()
}

func (c *Client) checkImplicitTLS(ctx context.Context) error {
	conn, stop, err := c.dialTLS(ctx)
	if err != nil {
		return err
	}
	defer stop()
	defer conn.Close()

	client, err := smtp.NewClient(conn, c.serverName)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Quit()
}

func (c *Client) sendStartTLS(ctx context.Context, from string, recipients []string, data []byte) error {
	conn, stop, err := c.dialPlain(ctx)
	if err != nil {
		return err
	}
	defer stop()
	defer conn.Close()

	client, err := smtp.NewClient(conn, c.serverName)
	if err != nil {
		return err
	}
	defer client.Close()

	if ok, _ := client.Extension(ExtensionStartTLS); !ok {
		return ErrMissingStartTLS
	}

	if err := client.StartTLS(c.tlsConfig.Clone()); err != nil {
		return err
	}

	return c.deliver(client, from, recipients, data)
}

func (c *Client) sendImplicitTLS(ctx context.Context, from string, recipients []string, data []byte) error {
	conn, stop, err := c.dialTLS(ctx)
	if err != nil {
		return err
	}
	defer stop()
	defer conn.Close()

	client, err := smtp.NewClient(conn, c.serverName)
	if err != nil {
		return err
	}
	defer client.Close()

	return c.deliver(client, from, recipients, data)
}

func (c *Client) deliver(client *smtp.Client, from string, recipients []string, data []byte) error {
	if c.User != "" || c.Password != "" {
		if ok, _ := client.Extension("AUTH"); !ok {
			return ErrMissingAuth
		}
		if err := client.Auth(smtp.PlainAuth("", c.User, c.Password, c.serverName)); err != nil {
			return err
		}
	}

	if err := client.Mail(from); err != nil {
		return err
	}
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	return client.Quit()
}

func (c *Client) dialPlain(ctx context.Context) (net.Conn, func(), error) {
	dialer := net.Dialer{Timeout: c.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", c.Address)
	if err != nil {
		return nil, nil, err
	}
	stop := c.bindConnContext(ctx, conn)
	return conn, stop, nil
}

func (c *Client) dialTLS(ctx context.Context) (net.Conn, func(), error) {
	dialer := &net.Dialer{Timeout: c.timeout}
	tlsDialer := &tls.Dialer{
		NetDialer: dialer,
		Config:    c.tlsConfig.Clone(),
	}
	conn, err := tlsDialer.DialContext(ctx, "tcp", c.Address)
	if err != nil {
		return nil, nil, err
	}
	stop := c.bindConnContext(ctx, conn)
	return conn, stop, nil
}

func (c *Client) bindConnContext(ctx context.Context, conn net.Conn) func() {
	if deadline, ok := c.connectionDeadline(ctx); ok {
		_ = conn.SetDeadline(deadline)
	}

	stopped := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-stopped:
		}
	}()

	return func() {
		close(stopped)
	}
}

func (c *Client) connectionDeadline(ctx context.Context) (time.Time, bool) {
	var deadline time.Time
	var ok bool

	if c.timeout > 0 {
		deadline = time.Now().Add(c.timeout)
		ok = true
	}

	if ctxDeadline, hasCtxDeadline := ctx.Deadline(); hasCtxDeadline && (!ok || ctxDeadline.Before(deadline)) {
		deadline = ctxDeadline
		ok = true
	}

	return deadline, ok
}
