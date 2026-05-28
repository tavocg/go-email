// Package mailer defines messages and delivery interfaces for email backends.
package mailer

import "context"

// Mailer sends email messages.
type Mailer interface {
	Send(ctx context.Context, message *Message) error
}
