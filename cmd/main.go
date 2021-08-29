package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v35/github"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github/jungwinter/codeowners"
)

func main() {
	log.SetFormatter(&log.JSONFormatter{
		DisableTimestamp: true,
		PrettyPrint:      false,
	})

	if err := inspect(); err != nil {
		log.WithError(err).Fatal("failed to inspect")
	}
}

func inspect() error {
	ctx := context.Background()
	// TODO: Support enterprise github client
	cli := codeowners.NewGitHubClient(ctx, "")

	owners, err := codeowners.Inspect(ctx, cli, "")
	if err != nil {
		return err
	}
	log.WithField("owners", owners).Info("finished to inspect")
	return nil
}

func replace() error {
	ctx := context.Background()
	// TODO: Support enterprise github client
	cli := codeowners.NewGitHubClient(ctx, "")
	// TODO: Pass by commandline argument
	repos, err := codeowners.ListActivatedRepositories(ctx, cli, "org")
	if err != nil {
		return err
	}

	for _, r := range repos {
		content, err := codeowners.GetCodeownersContent(ctx, cli, r)
		if errors.Cause(err) == codeowners.ErrNotFound {
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
		replaced := codeowners.ReplaceAll(s, o, n)
		if s == replaced {
			log.WithField("repo", r.GetName()).Info("no target owner")
			continue
		}

		log.WithField("repo", r.GetName()).WithField("after", replaced).Info("replaced")

		msg := github.String(fmt.Sprintf("Update %s to %s", o, n))
		if err := codeowners.CreatePatch(ctx, cli, r, content, replaced, msg); err != nil {
			return err
		}

		// TODO: Pass by commandline option (e.g. --pr-body "text")
		body := `
Update codeowners.
`
		// TODO: Pass by commandline option (e.g. --pr-title "text")
		if _, err := codeowners.OpenPR(ctx, cli, r, "pr title", "update-codeowners", body, &github.ReviewersRequest{
			// TODO: Request review "a", "b" either person or team
			Reviewers:     []string{"a"},
			TeamReviewers: []string{"b"},
		}); err != nil {
			return err
		}

		time.Sleep(3 * time.Second)
	}

	return nil
}
