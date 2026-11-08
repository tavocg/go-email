package smtp

import (
	"fmt"
	"net"
	stdsmtp "net/smtp"
	"net/textproto"
	"testing"
	"time"
)

func TestDeliverReportsDataAcceptance(t *testing.T) {
	for _, tt := range []struct {
		name, dataReply, quitReply string
		wantError                  bool
	}{
		{"accepted", "250 queued", "221 bye", false},
		{"accepted then disconnected", "250 queued", "", false},
		{"accepted then quit rejected", "250 queued", "500 quit rejected", false},
		{"data rejected", "550 message rejected", "", true},
		{"data acknowledgement missing", "", "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			conn, peer := net.Pipe()
			t.Cleanup(func() {
				conn.Close()
				peer.Close()
			})
			deadline := time.Now().Add(5 * time.Second)
			if err := conn.SetDeadline(deadline); err != nil {
				t.Fatal(err)
			}
			if err := peer.SetDeadline(deadline); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- serveDelivery(peer, tt.dataReply, tt.quitReply) }()

			client, err := stdsmtp.NewClient(conn, "localhost")
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			c := &Client{}
			err = c.deliver(client, "sender@example.com", []string{"recipient@example.com"}, []byte("Subject: test\r\n\r\nbody\r\n"))
			if (err != nil) != tt.wantError {
				t.Errorf("deliver() error = %v, want error = %v", err, tt.wantError)
			}
			if err := <-done; err != nil {
				t.Fatalf("SMTP fixture: %v", err)
			}
		})
	}
}

func serveDelivery(conn net.Conn, dataReply, quitReply string) error {
	defer conn.Close()
	wire := textproto.NewConn(conn)
	if err := wire.PrintfLine("220 localhost ready"); err != nil {
		return err
	}
	for _, step := range []struct{ command, response string }{
		{"EHLO localhost", "250 localhost"},
		{"MAIL FROM:<sender@example.com>", "250 sender accepted"},
		{"RCPT TO:<recipient@example.com>", "250 recipient accepted"},
		{"DATA", "354 send message"},
	} {
		command, err := wire.ReadLine()
		if err != nil {
			return err
		}
		if command != step.command {
			return fmt.Errorf("command = %q, want %q", command, step.command)
		}
		if err := wire.PrintfLine("%s", step.response); err != nil {
			return err
		}
	}
	data, err := wire.ReadDotBytes()
	if err != nil {
		return err
	}
	if string(data) != "Subject: test\n\nbody\n" {
		return fmt.Errorf("unexpected message data: %q", data)
	}
	if dataReply == "" {
		return nil
	}
	if err := wire.PrintfLine("%s", dataReply); err != nil {
		return err
	}
	if quitReply == "" {
		return nil
	}
	command, err := wire.ReadLine()
	if err != nil {
		return err
	}
	if command != "QUIT" {
		return fmt.Errorf("command = %q, want QUIT", command)
	}
	return wire.PrintfLine("%s", quitReply)
}
