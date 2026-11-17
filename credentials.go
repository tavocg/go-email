package email

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"time"
)

// CredentialsConfig identifies the SMTP service used to verify credentials.
// Address must include a port. TLS certificates are always verified against
// the address hostname. The default transport is implicit TLS (usually 465).
type CredentialsConfig struct {
	Address string
	// StartTLS requires upgrading to TLS before authentication (usually port 587).
	StartTLS bool
	// RootCAs optionally supplies trusted CAs; nil uses the system trust store.
	RootCAs *x509.CertPool
	// Timeout bounds the entire operation. Zero uses ten seconds.
	Timeout time.Duration
}

// VerifyCredentials authenticates with SMTP AUTH PLAIN over verified TLS,
// without sending a message. A nil error means the server accepted the credentials.
// Errors may indicate configuration, connectivity, TLS, or authentication failures;
// an error does not necessarily mean the password is incorrect.
func VerifyCredentials(ctx context.Context, config CredentialsConfig, user, password string) error {
	if ctx == nil {
		return errors.New("context must not be nil")
	}
	host, port, err := net.SplitHostPort(config.Address)
	if err != nil || host == "" || port == "" {
		return errors.New("SMTP address must be in host:port form")
	}
	if user == "" || password == "" {
		return errors.New("user and password must not be empty")
	}
	if config.Timeout < 0 {
		return errors.New("timeout must not be negative")
	}
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	tlsConfig := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12, RootCAs: config.RootCAs}
	dialer := &net.Dialer{}
	var conn net.Conn
	if config.StartTLS {
		conn, err = dialer.DialContext(ctx, "tcp", config.Address)
	} else {
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: tlsConfig}).DialContext(ctx, "tcp", config.Address)
	}
	if err != nil {
		return fmt.Errorf("connect to SMTP: %w", err)
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP greeting: %w", err)
	}
	defer client.Close()
	if config.StartTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("SMTP server does not advertise STARTTLS")
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("SMTP TLS: %w", err)
		}
	}
	if ok, _ := client.Extension("AUTH"); !ok {
		return errors.New("SMTP server does not advertise AUTH")
	}
	if err := client.Auth(smtp.PlainAuth("", user, password, host)); err != nil {
		return fmt.Errorf("SMTP authentication: %w", err)
	}
	// Authentication succeeded; a failure during QUIT does not invalidate it.
	_ = client.Quit()
	return nil
}
