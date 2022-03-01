package main

import (
	"context"
	"sort"
	"strings"

	"github.com/google/go-github/v42/github"
	"github.com/pkg/errors"
)

type Codeowner struct {
	Name     string
	OwnRepos []string
}

func Inspect(ctx context.Context, cli *github.Client, owner string) ([]*Codeowner, error) {
	users, err := listMemberNames(ctx, cli, owner)
	if err != nil {
		return nil, err
	}

	teams, err := listTeamNames(ctx, cli, owner)
	if err != nil {
		return nil, err
	}

	ownerMapByName, err := listAllCodeowners(ctx, cli, owner)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(ownerMapByName))
	for k := range ownerMapByName {
		names = append(names, k)
	}
	sort.Strings(names)

	diffNames := diff(names, append(users, teams...))

	owners := make([]*Codeowner, len(diffNames))
	for i, n := range diffNames {
		owners[i] = ownerMapByName[n]
	}
	return owners, nil
}

func listMemberNames(ctx context.Context, cli *github.Client, owner string) ([]string, error) {
	users, err := ListMembers(ctx, cli, owner)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(users))
	for i, user := range users {
		names[i] = user.GetLogin()
	}
	return names, nil
}

func listTeamNames(ctx context.Context, cli *github.Client, owner string) ([]string, error) {
	teams, err := ListTeams(ctx, cli, owner)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(teams))
	for i, team := range teams {
		names[i] = owner + "/" + team.GetSlug()
	}
	return names, nil
}

func listAllCodeowners(ctx context.Context, cli *github.Client, owner string) (map[string]*Codeowner, error) {
	rr, err := ListActivatedRepositories(ctx, cli, owner)
	if err != nil {
		return nil, err
	}

	ownersByRepo := make(map[string][]string, len(rr))
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

		ownersByRepo[r.GetName()] = parseCodeowners(s)
	}

	return groupByCodeowner(ownersByRepo), nil
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

func groupByCodeowner(ownersByRepo map[string][]string) map[string]*Codeowner {
	ownerMap := make(map[string]*Codeowner)
	for k, vv := range ownersByRepo {
		for _, v := range vv {
			if o, ok := ownerMap[v]; ok {
				o.OwnRepos = append(o.OwnRepos, k)
			} else {
				ownerMap[v] = &Codeowner{
					Name:     v,
					OwnRepos: []string{k},
				}
			}
		}
	}
	return ownerMap
}

func set(ss []string) []string {
	if len(ss) == 0 {
		return nil
	}
	if len(ss) == 1 {
		return ss
	}

	m := make(map[string]struct{})
	unique := make([]string, 0, len(ss))
	for _, s := range ss {
		if _, ok := m[s]; ok {
			continue
		}
		m[s] = struct{}{}
		unique = append(unique, s)
	}
	return unique
}

// diff returns result of `a` not in `b` case insensitive.
func diff(a, b []string) []string {
	m := make(map[string]struct{}, len(b))
	for _, k := range b {
		m[strings.ToLower(k)] = struct{}{}
	}

	unique := make([]string, 0, len(a))
	for _, k := range a {
		if _, ok := m[strings.ToLower(k)]; ok {
			continue
		}
		unique = append(unique, k)
	}
	return unique
}
