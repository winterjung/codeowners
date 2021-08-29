package codeowners

import (
	"context"
	"github.com/google/go-github/v35/github"
	"github.com/pkg/errors"
	"sort"
	"strings"
)

func Inspect(ctx context.Context, cli *github.Client, owner string) ([]string, error) {
	owners, err := listAllCodeowners(ctx, cli, owner)
	if err != nil {
		return nil, err
	}

	// TODO: Fetch current people in org
	// TODO: Fetch current teams in org
	// TODO: Diff
	return owners, nil
}

func listAllCodeowners(ctx context.Context, cli *github.Client, owner string) ([]string, error) {
	rr, err := ListActivatedRepositories(ctx, cli, owner)
	if err != nil {
		return nil, err
	}

	owners := make([][]string, 0, len(rr))
	for _, r := range rr {
		content, err := GetCodeownersContent(ctx, cli, r)
		if errors.Cause(err) == ErrNotFound {
			continue
		}
		if err != nil {
			return nil, err
		}

		s, err := content.GetContent()
		if err != nil {
			return nil, err
		}

		owners = append(owners, parseCodeowners(s))
	}
	return set(flatten(owners...)), nil
}

func parseCodeowners(s string) []string {
	ss := strings.Split(s, sep)
	nn := make([]string, 0)
	for _, l := range ss {
		if strings.HasPrefix(l, commentPrefix) {
			continue
		}
		if !strings.Contains(l, mentionPrefix) {
			continue
		}
		cc := strings.Split(l, mentionPrefix)
		for _, name := range cc[1:] {
			// identifier
			n := strings.TrimSpace(name)
			nn = append(nn, n)
		}
	}
	return set(nn)
}

func set(ss []string) []string {
	if len(ss) == 0 {
		return nil
	}

	m := make(map[string]struct{})
	for _, s := range ss {
		m[s] = struct{}{}
	}

	set := make([]string, 0, len(m))
	for s := range m {
		set = append(set, s)
	}
	sort.Strings(set)
	return set
}

func flatten(sss ...[]string) []string {
	ss := make([]string, 0)
	for _, elem := range sss {
		if len(elem) == 0 {
			continue
		}
		ss = append(ss, elem...)
	}
	return ss
}
