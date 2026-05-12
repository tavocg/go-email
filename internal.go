package email

// validAddress implements Normalize(email string) string, it is private to
// avoid using Normalize on non-valid email addresses.
type validAddress string

type errStr string

func (e errStr) Error() string {
	return string(e)
}
