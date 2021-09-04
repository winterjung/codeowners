package codeowners

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v35/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_listAllCodeowners(t *testing.T) {
	const (
		mockOwner = "some-org"
		mockRepo  = "some-repo"
		mockRepo2 = "some-repo-2"
	)

	cases := []struct {
		name       string
		expectFunc func(http.ResponseWriter, *http.Request)
		expected   []string
	}{
		{
			name: "no codeowner file",
			expectFunc: func(rw http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v3/orgs/%s/repos", mockOwner) {
					rw.WriteHeader(http.StatusOK)
					rw.Header().Set("Content-Type", "application/json")
					_, err := io.WriteString(rw, fmt.Sprintf(`[
	{
		"owner": {"login": "%s"},
		"name": "%s",
		"default_branch": "main"
	}
]`, mockOwner, mockRepo))
					require.NoError(t, err)
					return
				}
				if r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/branches/%s", mockOwner, mockRepo, prBranch) {
					rw.WriteHeader(http.StatusNotFound)
					return
				}
				if r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/contents/%s", mockOwner, mockRepo, ".github/CODEOWNERS") {
					rw.WriteHeader(http.StatusNotFound)
					return
				}
				if r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/contents/%s", mockOwner, mockRepo, "CODEOWNERS") {
					rw.WriteHeader(http.StatusNotFound)
					return
				}
				t.Errorf("%s, method: %s, request uri: %s", "should not reach here", r.Method, r.RequestURI)
			},
			expected: nil,
		},
		{
			name: "well merged",
			expectFunc: func(rw http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v3/orgs/%s/repos", mockOwner) {
					rw.WriteHeader(http.StatusOK)
					rw.Header().Set("Content-Type", "application/json")
					_, err := io.WriteString(rw, fmt.Sprintf(`[
	{
		"owner": {"login": "%s"},
		"name": "%s",
		"default_branch": "main"
	},
	{
		"owner": {"login": "%s"},
		"name": "%s",
		"default_branch": "main"
	}
]`, mockOwner, mockRepo, mockOwner, mockRepo2))
					require.NoError(t, err)
					return
				}
				if r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/branches/%s", mockOwner, mockRepo, prBranch) {
					rw.WriteHeader(http.StatusNotFound)
					return
				}
				if r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/branches/%s", mockOwner, mockRepo2, prBranch) {
					rw.WriteHeader(http.StatusNotFound)
					return
				}
				if r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/contents/%s", mockOwner, mockRepo, ".github/CODEOWNERS") {
					rw.WriteHeader(http.StatusOK)
					rw.Header().Set("Content-Type", "application/json")
					_, err := io.WriteString(rw, `{
  "content": "* @a\n.github @b @team/a"
}`)
					require.NoError(t, err)
					return
				}
				if r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/contents/%s", mockOwner, mockRepo2, ".github/CODEOWNERS") {
					rw.WriteHeader(http.StatusOK)
					rw.Header().Set("Content-Type", "application/json")
					_, err := io.WriteString(rw, `{
  "content": "* @b @c\n.github @a @c\n"
}`)
					require.NoError(t, err)
					return
				}
				t.Errorf("%s, method: %s, request uri: %s", "should not reach here", r.Method, r.RequestURI)
			},
			expected: []string{"a", "b", "c", "team/a"},
		},
		{
			name: "empty repos",
			expectFunc: func(rw http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet && r.RequestURI == fmt.Sprintf("/api/v3/orgs/%s/repos", mockOwner) {
					rw.WriteHeader(http.StatusOK)
					rw.Header().Set("Content-Type", "application/json")
					_, err := io.WriteString(rw, `[]`)
					require.NoError(t, err)
				}
			},
			expected: nil,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(
					tc.expectFunc,
				),
			)
			defer server.Close()

			mockGithubCli, err := github.NewEnterpriseClient(server.URL, server.URL, server.Client())
			require.NoError(t, err)

			ctx := context.Background()
			got, err := listAllCodeowners(ctx, mockGithubCli, mockOwner)

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}

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

func Test_flatten(t *testing.T) {
	cases := []struct {
		name     string
		given    [][]string
		expected []string
	}{
		{
			name:     "nil",
			given:    nil,
			expected: []string{},
		},
		{
			name:     "empty",
			given:    [][]string{{}},
			expected: []string{},
		},
		{
			name:     "one",
			given:    [][]string{{"a"}},
			expected: []string{"a"},
		},
		{
			name:     "many",
			given:    [][]string{{"a"}, {"b", "c"}, {"d"}},
			expected: []string{"a", "b", "c", "d"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := flatten(tc.given...)

			assert.Equal(t, tc.expected, got)
		})
	}
}

func Test_diff(t *testing.T) {
	cases := []struct {
		name     string
		a        []string
		b        []string
		expected []string
	}{
		{
			name:     "a - b",
			a:        []string{"a", "b"},
			b:        []string{"b", "c"},
			expected: []string{"a"},
		},
		{
			name:     "all diff",
			a:        []string{"a", "b"},
			b:        []string{"c", "d"},
			expected: []string{"a", "b"},
		},
		{
			name:     "equal",
			a:        []string{"a", "b"},
			b:        []string{"a", "b"},
			expected: []string{},
		},
		{
			name:     "case insensitive",
			a:        []string{"a", "B"},
			b:        []string{"A", "b"},
			expected: []string{},
		},
		{
			name:     "nil a",
			a:        nil,
			b:        []string{"b"},
			expected: []string{},
		},
		{
			name:     "nil b",
			a:        []string{"A", "b"},
			b:        nil,
			expected: []string{"A", "b"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := diff(tc.a, tc.b)

			assert.Equal(t, tc.expected, got)
		})
	}
}
