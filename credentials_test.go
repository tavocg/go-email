package email

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

func TestVerifyCredentials(t *testing.T) {
	certServer := httptest.NewTLSServer(nil)
	cert := certServer.TLS.Certificates[0]
	roots := x509.NewCertPool()
	roots.AddCert(certServer.Certificate())
	certServer.Close()
	for _, tc := range []struct {
		name                               string
		startTLS, untrusted, noTLS, reject bool
	}{
		{name: "implicit TLS"}, {name: "STARTTLS", startTLS: true},
		{name: "untrusted implicit TLS", untrusted: true},
		{name: "untrusted STARTTLS", startTLS: true, untrusted: true},
		{name: "missing STARTTLS", startTLS: true, noTLS: true},
		{name: "rejected password", reject: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			commands := make(chan []string, 1)
			go func() {
				var seen []string
				defer func() { commands <- seen }()
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
				config := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
				encrypted := !tc.startTLS
				if encrypted {
					conn = tls.Server(conn, config)
				}
				fmt.Fprint(conn, "220 test SMTP\r\n")
				reader := bufio.NewReader(conn)
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					line = strings.TrimSpace(line)
					seen = append(seen, line)
					switch {
					case strings.HasPrefix(line, "EHLO"):
						if !encrypted && !tc.noTLS {
							fmt.Fprint(conn, "250-test\r\n250-STARTTLS\r\n250 AUTH PLAIN\r\n")
						} else {
							fmt.Fprint(conn, "250-test\r\n250 AUTH PLAIN\r\n")
						}
					case line == "STARTTLS":
						fmt.Fprint(conn, "220 upgrade\r\n")
						conn = tls.Server(conn, config)
						encrypted = true
						reader = bufio.NewReader(conn)
					case strings.HasPrefix(line, "AUTH "):
						if !encrypted {
							return
						}
						if tc.reject {
							fmt.Fprint(conn, "535 invalid credentials\r\n")
						} else {
							fmt.Fprint(conn, "235 authenticated\r\n")
						}
					case line == "QUIT":
						fmt.Fprint(conn, "221 bye\r\n")
						return
					default:
						fmt.Fprint(conn, "500 unexpected command\r\n")
						return
					}
				}
			}()
			config := CredentialsConfig{Address: listener.Addr().String(), StartTLS: tc.startTLS, RootCAs: roots, Timeout: time.Second}
			if tc.untrusted {
				config.RootCAs = x509.NewCertPool()
			}
			err = VerifyCredentials(context.Background(), config, "user@example.com", "secret")
			wantErr := tc.untrusted || tc.noTLS || tc.reject
			if (err != nil) != wantErr {
				t.Fatalf("error = %v, want error %v", err, wantErr)
			}
			if tc.reject {
				var smtpErr *textproto.Error
				if !errors.As(err, &smtpErr) || smtpErr.Code != 535 {
					t.Fatalf("expected SMTP 535, got %v", err)
				}
			}
			seen := <-commands
			auth := false
			for _, command := range seen {
				if strings.HasPrefix(command, "AUTH ") {
					auth = true
					expected := "AUTH PLAIN " + base64.StdEncoding.EncodeToString([]byte("\x00user@example.com\x00secret"))
					if command != expected {
						t.Error("unexpected authentication payload")
					}
				}
				if strings.HasPrefix(command, "MAIL ") || strings.HasPrefix(command, "RCPT ") || command == "DATA" {
					t.Error("attempted message delivery")
				}
			}
			if auth != !(tc.untrusted || tc.noTLS) {
				t.Fatalf("authentication attempted = %v", auth)
			}
		})
	}
}
