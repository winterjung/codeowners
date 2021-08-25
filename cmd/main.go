package main

import (
	"context"
	"time"

	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"

	"github/jungwinter/codeowners"
)

func main() {
	log.SetFormatter(&log.JSONFormatter{
		DisableTimestamp: true,
		PrettyPrint:      false,
	})

	if err := run(); err != nil {
		log.WithError(err).Fatal("failed to run")
	}
}

func run() error {
	ctx := context.Background()
	cli := codeowners.NewGitHubClient(ctx, "")
	repos, err := codeowners.ListActivatedRepositories(ctx, cli, "orgname")
	if err != nil {
		return err
	}

	// temp
	//r, _, _ := cli.Repositories.Get(ctx, "orgname", "reponame")
	//repos := []*github.Repository{r}
	for _, r := range repos {
		content, err := codeowners.GetCodeownersContent(ctx, cli, r)
		if errors.Cause(err) == codeowners.ErrNotFound {
			log.WithField("repo", r.GetName()).Info("passed because codeowner file is not exist")
			continue
		}
		if err != nil {
			return err
		}

		s, err := content.GetContent()
		if err != nil {
			return err
		}

		o, n := "a", "b"
		replaced := codeowners.ReplaceAll(s, o, n)
		if s == replaced {
			log.WithField("repo", r.GetName()).Info("passed because target owner is not exist")
			continue
		}

		log.WithField("repo", r.GetName()).WithField("after", replaced).Info("replaced")

		//msg := github.String(fmt.Sprintf("Update %s to %s", o, n))
		//if err := codeowners.CreatePatch(ctx, cli, r, content, replaced, msg); err != nil {
		//	return err
		//}
		time.Sleep(3 * time.Second)
	}

	return nil
}
