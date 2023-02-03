package txtsanitize

import "testing"

func TestTXTSanitize(t *testing.T) {
	for _, tst := range []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "embedded double quoted string",
			input:    `asdf " jkl "`,
			expected: `"asdf \" jkl \""`,
		},
		{
			name:     "pre-escaped embedded double quote",
			input:    `"i know what i'm doing \" right there"`,
			expected: `"i know what i'm doing \" right there"`,
		},
		{
			name:     "quoted escaped slash with raw double quote",
			input:    `"i don't know what i'm doing \\" right there"`,
			expected: `"i don't know what i'm doing \\\" right there"`,
		},
		{
			name:     "unicode sequence unmolested with embedded quotes",
			input:    `"ç" is equal to "\195\167"`,
			expected: `"\"ç\" is equal to \"\195\167\""`,
		},
		{
			name:     "beginning and ending quoted sequences",
			input:    `"foo" and other stuff "bar"`,
			expected: `"\"foo\" and other stuff \"bar\""`,
		},
		{
			name:     "empty",
			input:    ``,
			expected: `""`,
		},
		{
			name:     "escapeception",
			input:    `this \" is escaped, this \\" isn't, but this \\\" is, but this \\\\" isn't`,
			expected: `"this \" is escaped, this \\\" isn't, but this \\\" is, but this \\\\\" isn't"`,
		},
		{
			name:     "starts with quoted section",
			input:    `"this is quoted" but the rest isn't`,
			expected: `"\"this is quoted\" but the rest isn't"`,
		},
		{
			name:     "ends with quoted section",
			input:    `only the "end is quoted"`,
			expected: `"only the \"end is quoted\""`,
		},
	} {
		t.Run(tst.name, func(t *testing.T) {
			out := TXTSanitize(tst.input)
			if out != tst.expected {
				t.Errorf("failed: expected %s got %s", tst.expected, out)
			}
			// make sure sanitized output is idempotent
			recycled := TXTSanitize(out)
			if out != recycled {
				t.Errorf("identity test failed: expected %s got %s", out, recycled)
			}
		})

	}
}
