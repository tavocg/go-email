package smtp

type errString string

func (e errString) Error() string {
	return string(e)
}

const (
	ErrInvalidAddress     errString = "smtp address must be in host:port form"
	ErrMessageMissingBody errString = "message must have a body or at least one attachment"
	ErrMessageMissingFrom errString = "message must have a from address"
	ErrMissingAuth        errString = "smtp server does not advertise AUTH"
	ErrMissingRecipients  errString = "message must have at least one recipient"
	ErrMissingStartTLS    errString = "smtp server does not advertise STARTTLS"
	ErrNilContext         errString = "context must not be nil"
	ErrNilMessage         errString = "message is nil"
)
