package codeowners

import (
	"context"
	"net/http"

	"github.com/google/go-github/v35/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

const (
	prBranch = "update-codeowners"
)

var (
	ErrNotFound = errors.New("not found")
)

func NewGitHubClient(ctx context.Context, token string) *github.Client {
	var httpClient *http.Client

	if token != "" {
		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: token,
		})
		httpClient = oauth2.NewClient(ctx, tokenSource)
	}
	return github.NewClient(httpClient)
}

func ListActivatedRepositories(ctx context.Context, cli *github.Client, owner string) ([]*github.Repository, error) {
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

func GetCodeownersContent(ctx context.Context, cli *github.Client, r *github.Repository, ref *string) (*github.RepositoryContent, error) {
	fc, err := getContent(ctx, cli, r, ".github/CODEOWNERS", ref)
	if err != nil {
		if errors.Cause(err) == ErrNotFound {
			fc, err := getContent(ctx, cli, r, "CODEOWNERS", ref)
			if err != nil {
				return nil, err
			}
			return fc, nil
		}
		return nil, err
	}
	return fc, nil
}

func CreatePatch(ctx context.Context, cli *github.Client, r *github.Repository, old *github.RepositoryContent, newContent string) error {
	var (
		owner = r.GetOwner().GetLogin()
		name  = r.GetName()
	)
	exist, err := isBranchExists(cli, ctx, r, prBranch)
	if err != nil {
		return err
	}
	if !exist {
		mainRef, _, err := cli.Git.GetRef(ctx, owner, name, "refs/heads/"+r.GetDefaultBranch())
		if err != nil {
			return errors.WithStack(err)
		}

		prRef := &github.Reference{
			Ref: github.String("refs/heads/" + prBranch),
			Object: &github.GitObject{
				SHA: mainRef.Object.SHA,
			},
		}

		if _, _, err := cli.Git.CreateRef(ctx, owner, name, prRef); err != nil {
			return errors.WithStack(err)
		}
	}

	opt := &github.RepositoryContentFileOptions{
		Message: github.String("Update codeowners"),
		Content: []byte(newContent),
		SHA:     github.String(old.GetSHA()),
		Branch:  github.String(prBranch),
	}
	if _, _, err := cli.Repositories.CreateFile(ctx, owner, name, old.GetPath(), opt); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func OpenPR(ctx context.Context, cli *github.Client, r *github.Repository, prTitle, head, body string) (*github.PullRequest, error) {
	req := &github.NewPullRequest{
		Title: github.String(prTitle),
		Head:  github.String(head),
		Base:  r.DefaultBranch,
		Body:  github.String(body),
		Draft: github.Bool(false),
	}
	pr, _, err := cli.PullRequests.Create(ctx, r.GetOwner().GetLogin(), r.GetName(), req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return pr, nil
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

func getContent(ctx context.Context, cli *github.Client, r *github.Repository, path string, ref *string) (*github.RepositoryContent, error) {
	opt := &github.RepositoryContentGetOptions{
		Ref: r.GetDefaultBranch(),
	}
	if ref != nil {
		opt.Ref = *ref
	}

	fc, _, res, err := cli.Repositories.GetContents(ctx, r.GetOwner().GetLogin(), r.GetName(), path, opt)
	if err != nil {
		if res != nil && res.StatusCode == http.StatusNotFound {
			return nil, errors.WithStack(ErrNotFound)
		}
		return nil, errors.WithStack(err)
	}

	return fc, nil
}
