package main

import (
	"strings"
)

const (
	sep = "\n"

	commentPrefix = "#"
	mentionPrefix = "@"
)

// ReplaceAll returns the string s of old replaced by new in multilines content.
func ReplaceAll(s, old, new string) string {
	ss := strings.Split(s, sep)
	ll := make([]string, len(ss))
	for i, l := range ss {
		ll[i] = Replace(l, old, new)
	}

	return strings.Join(ll, sep)
}

// Replace returns the string s of old replaced by new in a single line.
func Replace(s, old, new string) string {
	if strings.HasPrefix(s, commentPrefix) {
		return s
	}
	if !strings.Contains(s, mentionPrefix) {
		return s
	}

	old = strings.ToLower(old)
	if !strings.Contains(strings.ToLower(s), old) {
		return s
	}

	m := make(map[string]struct{})
	cc := strings.Split(s, mentionPrefix)
	stack, cands := []string{cc[0]}, cc[1:]
	for _, name := range cands {
		// identifier
		n := strings.TrimSpace(strings.ToLower(name))

		// non target first seen owner
		if _, ok := m[n]; n != old && !ok {
			m[n] = struct{}{}
			stack = append(stack, name)
			continue
		}

		replaced := strings.ReplaceAll(strings.ToLower(name), old, new)
		n = strings.TrimSpace(strings.ToLower(replaced))
		if n == "" {
			continue
		}
		if _, ok := m[n]; ok {
			continue
		}

		m[n] = struct{}{}
		stack = append(stack, replaced)
	}

	// remove trailing whitespace
	stack[len(stack)-1] = strings.TrimSpace(stack[len(stack)-1])
	return strings.Join(stack, mentionPrefix)
}
