package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v48/github"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetFormatter(&log.JSONFormatter{
		DisableTimestamp: true,
		PrettyPrint:      false,
	})

	if err := inspect(); err != nil {
		log.WithError(err).Fatal("failed to inspect")
	}
	if err := replace(); err != nil {
		log.WithError(err).Fatal("failed to replace")
	}
}

func inspect() error {
	ctx := context.Background()
	// TODO: Support enterprise github client
	cli := NewGitHubClient(ctx, "")

	owners, err := Inspect(ctx, cli, "org")
	if err != nil {
		return err
	}
	for _, o := range owners {
		log.WithField("owner", o.Name).WithField("repos", o.OwnRepos).Info("should be replaced")
	}
	return nil
}

func replace() error {
	ctx := context.Background()
	// TODO: Support enterprise github client
	cli := NewGitHubClient(ctx, "")
	// TODO: Pass by commandline argument
	repos, err := ListActivatedRepositories(ctx, cli, "org")
	if err != nil {
		return err
	}

	// TODO: Pass by commandline argument
	allowlist := map[string]struct{}{
		"repo": {},
	}

	// TODO: Pass by commandline argument
	denylist := map[string]struct{}{
		"repo": {},
	}
	for _, r := range repos {
		if _, ok := denylist[r.GetName()]; ok {
			log.WithField("repo", r.GetName()).Info("denied")
			continue
		}
		if _, ok := allowlist[r.GetName()]; !ok {
			log.WithField("repo", r.GetName()).Info("denied")
			continue
		}
		content, err := GetCodeownersContent(ctx, cli, r)
		if errors.Cause(err) == ErrNotFound {
			log.WithField("repo", r.GetName()).Info("no codeowner file")
			continue
		}
		if err != nil {
			return err
		}

		s, err := content.GetContent()
		if err != nil {
			return err
		}

		// TODO: Pass by commandline argument
		o, n := "a", "b"
		replaced := ReplaceAll(s, o, n)
		if s == replaced {
			log.WithField("repo", r.GetName()).Info("no target owner")
			continue
		}

		log.WithField("repo", r.GetName()).WithField("after", replaced).Info("replaced")

		msg := github.String(fmt.Sprintf("Update %s to %s", o, n))
		// TODO: Support remove command
		if n == "" {
			msg = github.String(fmt.Sprintf("Remove %s", o))
		}
		if err := CreatePatch(ctx, cli, r, content, replaced, msg); err != nil {
			return err
		}

		// TODO: Pass by commandline option (e.g. --pr-body "text")
		body := `
Update codeowners.
`
		// TODO: Pass by commandline option (e.g. --pr-title "text")
		if _, err := OpenPR(ctx, cli, r, "pr title", "update-codeowners", body, &github.ReviewersRequest{
			// TODO: Request review "a", "b" either person or team
			Reviewers:     []string{"a"},
			TeamReviewers: []string{"b"},
		}); err != nil {
			return err
		}
		log.WithField("repo", r.GetName()).Info("pr is opened")

		time.Sleep(3 * time.Second)
	}

	return nil
}
