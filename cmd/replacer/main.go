package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/google/go-github/v35/github"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const (
	commonCodeownerFilePath = ".github/CODEOWNERS"
	prBranch                = "update-codeowners"
)

var (
	errNotfound = errors.New("not found")
)

func main() {
	ctx := context.Background()
	cli := newGitHubClient(ctx, "")

	rr, err := listActivatedRepositories(ctx, cli, "org")
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.WithField("repo_count", len(rr)).Info("list repos")

	if _, err := os.Stat("output/org"); os.IsNotExist(err) {
		logrus.Info("update latest codeowners")
		if err := saveLatestCodeOwners(ctx, cli, rr); err != nil {
			logrus.Fatal(err)
		}
	} else {
		logrus.Info("use already exist codeowners")
	}
	//if err := list(ctx, cli, "org", rr); err != nil {
	//	logrus.Fatal(err)
	//}
	if err := open(ctx, cli, "org", rr); err != nil {
		logrus.Fatal(err)
	}
}

func saveLatestCodeOwners(ctx context.Context, cli *github.Client, rr []*github.Repository) error {
	for _, r := range rr {
		fc, err := getContent(ctx, cli, r, commonCodeownerFilePath)
		if errors.Cause(err) == errNotfound {
			logrus.WithField("repo", r.GetFullName()).Info("not found codeowner file")
			continue
		}
		if err != nil {
			return err
		}

		c, err := fc.GetContent()
		if err != nil {
			return errors.WithStack(err)
		}
		if err := write(fmt.Sprintf("output/%s", r.GetFullName()), c); err != nil {
			return err
		}
		logrus.WithField("repo", r.GetFullName()).Info("saved codeowner file")
	}

	return nil
}

func newGitHubClient(ctx context.Context, token string) *github.Client {
	var httpClient *http.Client

	if token != "" {
		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: token,
		})
		httpClient = oauth2.NewClient(ctx, tokenSource)
	}
	return github.NewClient(httpClient)
}

func listActivatedRepositories(ctx context.Context, cli *github.Client, owner string) ([]*github.Repository, error) {
	opt := &github.RepositoryListByOrgOptions{
		Type: "private",
		ListOptions: github.ListOptions{
			Page: 0,
		},
	}

	var allRepos []*github.Repository
	for {
		rr, resp, err := cli.Repositories.ListByOrg(ctx, owner, opt)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		allRepos = append(allRepos, rr...)

		if resp.NextPage == 0 {
			break
		}

		opt.ListOptions.Page = resp.NextPage
	}

	filtered := make([]*github.Repository, 0, len(allRepos))
	for _, r := range allRepos {
		if r.GetArchived() {
			continue
		}
		filtered = append(filtered, r)
	}

	return filtered, nil
}

func getContent(ctx context.Context, cli *github.Client, r *github.Repository, path string) (*github.RepositoryContent, error) {
	opt := &github.RepositoryContentGetOptions{
		Ref: r.GetDefaultBranch(),
	}

	fc, _, res, err := cli.Repositories.GetContents(ctx, r.GetOwner().GetLogin(), r.GetName(), path, opt)
	if err != nil {
		if res != nil && res.StatusCode == http.StatusNotFound {
			return nil, errors.WithStack(errNotfound)
		}
		return nil, errors.WithStack(err)
	}

	return fc, nil
}

func write(path, text string) error {
	f, err := os.Create(path)
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.Error(errors.WithStack(err))
		}
	}()

	_, err = f.WriteString(text)
	if err != nil {
		return errors.WithStack(err)
	}

	if err := f.Sync(); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

var (
	excludedRepoMap = map[string]bool{}
)

func list(ctx context.Context, cli *github.Client, owner string, rr []*github.Repository) error {
	files, err := ioutil.ReadDir(fmt.Sprintf("diff/%s", owner))
	if err != nil {
		return errors.WithStack(err)
	}

	repoMap := make(map[string]*github.Repository, len(rr))
	for _, r := range rr {
		repoMap[r.GetName()] = r
	}

	for _, f := range files {
		name := f.Name()
		if ok := excludedRepoMap[name]; ok {
			logrus.WithField("repo", name).Info("skipped")
			continue
		}

		newContent, err := ioutil.ReadFile(fmt.Sprintf("diff/%s/%s", owner, name))
		if err != nil {
			return errors.WithStack(err)
		}

		r, ok := repoMap[name]
		if !ok {
			return errors.Errorf("repe not found: %s", name)
		}

		fc, err := getContent(ctx, cli, r, commonCodeownerFilePath)
		if err != nil {
			return err
		}

		if err := createCodeownersPatch(ctx, cli, r, fc.GetSHA(), string(newContent)); err != nil {
			return err
		}

		logrus.WithField("repo", name).Info("created ref")
	}
	return nil
}

func createCodeownersPatch(ctx context.Context, cli *github.Client, r *github.Repository, oldContentSHA, newContent string) error {
	exist, err := isBranchExists(cli, ctx, r, prBranch)
	if err != nil {
		return err
	}
	if !exist {
		mainRef, _, err := cli.Git.GetRef(ctx, r.GetOwner().GetLogin(), r.GetName(), fmt.Sprintf("refs/heads/%s", r.GetDefaultBranch()))
		if err != nil {
			return errors.WithStack(err)
		}

		prRef := &github.Reference{
			Ref: github.String("refs/heads/" + prBranch),
			Object: &github.GitObject{
				SHA: mainRef.Object.SHA,
			},
		}

		_, _, err = cli.Git.CreateRef(ctx, r.GetOwner().GetLogin(), r.GetName(), prRef)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	opt := &github.RepositoryContentFileOptions{
		Message: github.String("Update codeowners"),
		Content: []byte(newContent),
		SHA:     github.String(oldContentSHA),
		Branch:  github.String(prBranch),
	}
	_, _, err = cli.Repositories.CreateFile(ctx, r.GetOwner().GetLogin(), r.GetName(), commonCodeownerFilePath, opt)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func isBranchExists(cli *github.Client, ctx context.Context, r *github.Repository, branch string) (bool, error) {
	_, res, err := cli.Repositories.GetBranch(ctx, r.GetOwner().GetLogin(), r.GetName(), branch)
	if err != nil {
		if res != nil && res.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, errors.WithStack(err)
	}

	return true, nil
}

func open(ctx context.Context, cli *github.Client, owner string, rr []*github.Repository) error {
	body := `
Updated codeowners.
This pr was generated by script. Please merge directly after review.

`
	files, err := ioutil.ReadDir(fmt.Sprintf("diff/%s", owner))
	if err != nil {
		return errors.WithStack(err)
	}

	repoMap := make(map[string]*github.Repository, len(rr))
	for _, r := range rr {
		repoMap[r.GetName()] = r
	}

	for _, f := range files {
		name := f.Name()
		if ok := excludedRepoMap[name]; ok {
			logrus.WithField("repo", name).Info("skipped")
			continue
		}

		r, ok := repoMap[name]
		if !ok {
			return errors.Errorf("repe not found: %s", name)
		}

		if _, err := openPR(ctx, cli, r, "Update codeowners", prBranch, body); err != nil {
			return err
		}
		logrus.WithField("repo", name).Info("opened pr")
		time.Sleep(time.Duration(3) * time.Second)
	}
	return nil
}

func openPR(ctx context.Context, cli *github.Client, r *github.Repository, prTitle, head, body string) (*github.PullRequest, error) {
	npr := &github.NewPullRequest{
		Title: github.String(prTitle),
		Head:  github.String(head),
		Base:  r.DefaultBranch,
		Body:  github.String(body),
		Draft: github.Bool(false),
	}
	pr, _, err := cli.PullRequests.Create(ctx, r.GetOwner().GetLogin(), r.GetName(), npr)
	if err != nil {
		return nil, err
	}
	return pr, nil
}
