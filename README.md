# go-email

`go-email` provides conservative email address validation, normalization helpers,
MIME message rendering, and small delivery backends.

The root module is:

```sh
go get github.com/tavocg/go-email
```

The Mailgun backend is published as a separate module:

```sh
go get github.com/tavocg/go-email/mailer/backends/mailgun
```

## Requirements

This project requires Go 1.24.0 or newer.

## Email Validation

Use `StrictParser` when you want a conservative ASCII-only address format for
storage, account identifiers, or comparisons.

```go
address, err := email.StrictParser("User.Name+signup@Example.com")
if err != nil {
	return err
}

address.Normalize()
fmt.Println(address.Address()) // user.name+signup@example.com
```

The strict parser trims surrounding whitespace and rejects addresses outside the
package's documented format. It does not check DNS, deliverability, real TLDs, or
provider-specific address rules.

`Normalize` lowercases the full address. Apply it consistently anywhere
addresses are stored or compared.

Plus-tag stripping is provider-specific, so it is explicit:

```go
address.Normalize(email.StripPlusTag())
```

## Blacklist Checks

`ValidAddress.IsBlacklisted` checks the address domain against a custom list or,
when no list is provided, the embedded disposable-domain list.

```go
blocked := address.IsBlacklisted([]string{"example.com"})
```

## MIME Messages

The `mailer` package builds MIME bytes suitable for SMTP `DATA` or HTTP API
backends.

```go
message := mailer.NewAlternativeMessage(
	"sender@example.com",
	[]string{"recipient@example.com"},
	"Welcome",
	"Welcome to the service.",
	"<p>Welcome to the service.</p>",
)
message.AttachWithType("hello.txt", "text/plain", []byte("hello\n"))

data, err := message.Bytes()
if err != nil {
	return err
}
```

Message rendering rejects CR or LF characters in header and envelope fields and
returns `mailer.ErrInvalidHeader`.

## SMTP

The SMTP backend supports implicit TLS and STARTTLS. `NewClient` validates the
address and configures TLS without touching the network. Use `CheckTransport`
when you want an explicit startup probe.

```go
client, err := smtpbackend.NewClient(
	"smtp.example.com:587",
	"username",
	"password",
	smtpbackend.WithStartTLS(),
)
if err != nil {
	return err
}

if err := client.CheckTransport(ctx); err != nil {
	return err
}

err = client.Send(ctx, message)
```

## Mailgun

The Mailgun backend uses Mailgun's HTTP API and accepts rendered MIME messages
from the shared `mailer.Message` type.

```go
client, err := mailgun.NewClient("mg.example.com", apiKey)
if err != nil {
	return err
}

err = client.Send(ctx, message)
```

## Examples

Runnable examples are available under `examples/`:

- `examples/validation`
- `examples/message`
- `examples/smtp`

## License

GNU General Public License v3.0. See `LICENSE`.
