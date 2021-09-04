package codeowners

import (
	"context"
	"sort"
	"strings"

	"github.com/google/go-github/v35/github"
	"github.com/pkg/errors"
)

func Inspect(ctx context.Context, cli *github.Client, owner string) ([]string, error) {
	owners, err := listAllCodeowners(ctx, cli, owner)
	if err != nil {
		return nil, err
	}

	users, err := ListMembers(ctx, cli, owner)
	if err != nil {
		return nil, err
	}
	userNames := make([]string, len(users))
	for i, user := range users {
		userNames[i] = user.GetLogin()
	}

	teams, err := ListTeams(ctx, cli, owner)
	if err != nil {
		return nil, err
	}
	teamNames := make([]string, len(teams))
	for i, team := range teams {
		teamNames[i] = team.GetSlug()
	}

	return diff(owners, append(userNames, teamNames...)), nil
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

func diff(a, b []string) []string {
	m := make(map[string]string)
	for _, k := range a {
		m[strings.ToLower(k)] = k
	}

	for _, k := range b {
		k = strings.ToLower(k)
		if _, ok := m[k]; ok {
			delete(m, k)
		}
	}

	d := make([]string, 0, len(m))
	for _, v := range m {
		d = append(d, v)
	}
	sort.Strings(d)
	return d
}
