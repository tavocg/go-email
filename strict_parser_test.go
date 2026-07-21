package email

import "testing"

func TestStrictParserAcceptsConservativeAddresses(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "basic",
			input: "user@example.com",
			want:  "user@example.com",
		},
		{
			name:  "trims surrounding whitespace",
			input: "  User.Name+tag@example.co  ",
			want:  "User.Name+tag@example.co",
		},
		{
			name:  "subdomain",
			input: "first_last@example.mail.co",
			want:  "first_last@example.mail.co",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := StrictParser(test.input)
			if err != nil {
				t.Fatalf("StrictParser(%q) error = %v, want nil", test.input, err)
			}
			if got == nil {
				t.Fatalf("StrictParser(%q) address = nil, want %q", test.input, test.want)
			}
			if got.Address() != test.want {
				t.Fatalf("StrictParser(%q) address = %q, want %q", test.input, got.Address(), test.want)
			}
		})
	}
}

func TestStrictParserRejectsInvalidAddresses(t *testing.T) {
	tests := []string{
		"",
		"missing-at.example.com",
		"user@@example.com",
		".user@example.com",
		"user.@example.com",
		"user..name@example.com",
		"user@example",
		"user@example..com",
		"user@-example.com",
		"user@example.c",
		"user@example.123",
		"user@example.com\r\nX-Injected: yes",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			got, err := StrictParser(input)
			if err != StrictParserError {
				t.Fatalf("StrictParser(%q) error = %v, want %v", input, err, StrictParserError)
			}
			if got != nil {
				t.Fatalf("StrictParser(%q) address = %q, want nil", input, got.Address())
			}
		})
	}
}
