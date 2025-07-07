package mailer

import (
	"testing"

	"github.com/linkly-id/auth/internal/conf"
	"github.com/stretchr/testify/require"
)

func TestTemplateHeaders(t *testing.T) {
	cases := []struct {
		from string
		typ  string
		exp  map[string][]string
	}{
		{
			from: `{"x-linkly-project-ref": ["abcjrhohrqmvcpjpsyzc"]}`,
			typ:  "OTHER-TYPE",
			exp: map[string][]string{
				"x-linkly-project-ref": {"abcjrhohrqmvcpjpsyzc"},
			},
		},

		{
			from: `{"X-Test-A": ["test-a", "test-b"], "X-Test-B": ["test-c", "abc $messageType"]}`,
			typ:  "TEST-MESSAGE-TYPE",
			exp: map[string][]string{
				"X-Test-A": {"test-a", "test-b"},
				"X-Test-B": {"test-c", "abc TEST-MESSAGE-TYPE"},
			},
		},

		{
			from: `{"X-Test-A": ["test-a", "test-b"], "X-Test-B": ["test-c", "abc $messageType"]}`,
			typ:  "OTHER-TYPE",
			exp: map[string][]string{
				"X-Test-A": {"test-a", "test-b"},
				"X-Test-B": {"test-c", "abc OTHER-TYPE"},
			},
		},

		{
			from: `{"X-Test-A": ["test-a", "test-b"], "X-Test-B": ["test-c", "abc $messageType"], "x-linkly-project-ref": ["abcjrhohrqmvcpjpsyzc"]}`,
			typ:  "OTHER-TYPE",
			exp: map[string][]string{
				"X-Test-A":               {"test-a", "test-b"},
				"X-Test-B":               {"test-c", "abc OTHER-TYPE"},
				"x-linkly-project-ref": {"abcjrhohrqmvcpjpsyzc"},
			},
		},
	}
	for _, tc := range cases {
		mailer := TemplateMailer{
			Config: &conf.GlobalConfiguration{
				SMTP: conf.SMTPConfiguration{
					Headers: tc.from,
				},
			},
		}
		require.NoError(t, mailer.Config.SMTP.Validate())

		hdrs := mailer.Headers(tc.typ)
		require.Equal(t, hdrs, tc.exp)
	}
}
