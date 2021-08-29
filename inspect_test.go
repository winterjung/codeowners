package codeowners

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_parseCodeowners(t *testing.T) {

	cases := []struct {
		name     string
		given    string
		expected []string
	}{
		{
			name:     "empty string",
			given:    "",
			expected: nil,
		},
		{
			name:     "one line",
			given:    "* @a @b @org/team",
			expected: []string{"a", "b", "org/team"},
		},
		{
			name:     "comment line",
			given:    "# @a @b",
			expected: nil,
		},
		{
			name:     "explicit no codeowners file",
			given:    ".github",
			expected: nil,
		},
		{
			name:     "duplicates",
			given:    "* @a\n.github @a",
			expected: []string{"a"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := parseCodeowners(tc.given)

			assert.Equal(t, tc.expected, got)
		})
	}
}

func Test_set(t *testing.T) {
	cases := []struct {
		name     string
		given    []string
		expected []string
	}{
		{
			name:     "nil",
			given:    nil,
			expected: nil,
		},
		{
			name:     "empty strings",
			given:    []string{"", ""},
			expected: []string{""},
		},
		{
			name:     "unique",
			given:    []string{"a", "b"},
			expected: []string{"a", "b"},
		},
		{
			name:     "duplicates",
			given:    []string{"b", "a", "c", "a"},
			expected: []string{"a", "b", "c"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := set(tc.given)

			assert.Equal(t, tc.expected, got)
		})
	}
}
