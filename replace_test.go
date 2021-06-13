package codeowners

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplaceAll(t *testing.T) {
	cases := []struct {
		name     string
		s        string
		old      string
		new      string
		expected string
	}{
		{
			name:     "multiline",
			s:        "* @a\na @a @b",
			old:      "a",
			new:      "b",
			expected: "* @b\na @b",
		},
		{
			name:     "ignore non rule line",
			s:        "# codeowners\n* @a\n\n",
			old:      "a",
			new:      "b",
			expected: "# codeowners\n* @b\n\n",
		},
		{
			name:     "ignore commented line",
			s:        "* @a\n# .github @a",
			old:      "a",
			new:      "b",
			expected: "* @b\n# .github @a",
		},
		{
			name:     "keep whitespace path name",
			s:        "* @a\n/example\\ path/ @a",
			old:      "a",
			new:      "b",
			expected: "* @b\n/example\\ path/ @b",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := ReplaceAll(tc.s, tc.old, tc.new)

			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestReplace(t *testing.T) {
	cases := []struct {
		name     string
		s        string
		old      string
		new      string
		expected string
	}{
		{
			name:     "a to b",
			s:        "* @a",
			old:      "a",
			new:      "b",
			expected: "* @b",
		},
		{
			name:     "keep priority",
			s:        "* @b @c @a",
			old:      "a",
			new:      "b",
			expected: "* @b @c",
		},
		{
			name:     "promote to keep priority",
			s:        "* @a @c @b",
			old:      "a",
			new:      "b",
			expected: "* @b @c",
		},
		{
			name:     "distinguish team",
			s:        "* @a/a @a @b",
			old:      "a/a",
			new:      "b",
			expected: "* @b @a",
		},
		{
			name:     "distinguish member",
			s:        "* @a/a @a @b",
			old:      "a",
			new:      "b",
			expected: "* @a/a @b",
		},
		{
			name:     "match exactly",
			s:        "* @a @aa",
			old:      "a",
			new:      "b",
			expected: "* @b @aa",
		},
		{
			name:     "keep whitespaces",
			s:        "*    @a",
			old:      "a",
			new:      "b",
			expected: "*    @b",
		},
		{
			name:     "keep all kind whitespace",
			s:        "*\t@a  @b\t\t@c",
			old:      "a",
			new:      "b",
			expected: "*\t@b  @c",
		},
		{
			name:     "remove trailing whitespace",
			s:        "* @a ",
			old:      "a",
			new:      "b",
			expected: "* @b",
		},
		{
			name:     "merge duplicates",
			s:        "* @a @a @a",
			old:      "a",
			new:      "b",
			expected: "* @b",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := Replace(tc.s, tc.old, tc.new)

			assert.Equal(t, tc.expected, got)
		})
	}
}
