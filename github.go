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
			return nil, errors.Wrap(err, "cli.Repositories.ListByOrg")
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

func GetCodeownersContent(ctx context.Context, cli *github.Client, r *github.Repository) (*github.RepositoryContent, error) {
	// If codeowner updating branch is already exist, use it's ref
	exist, err := isBranchExists(cli, ctx, r, prBranch)
	if err != nil {
		return nil, err
	}

	var ref *string
	if exist {
		ref = github.String("refs/heads/" + prBranch)
	}

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

func CreatePatch(ctx context.Context, cli *github.Client, r *github.Repository, old *github.RepositoryContent, newContent string, commitMsg *string) error {
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
			return errors.Wrap(err, "cli.Git.GetRef")
		}

		prRef := &github.Reference{
			Ref: github.String("refs/heads/" + prBranch),
			Object: &github.GitObject{
				SHA: mainRef.Object.SHA,
			},
		}

		if _, _, err := cli.Git.CreateRef(ctx, owner, name, prRef); err != nil {
			return errors.Wrap(err, "cli.Git.CreateRef")
		}
	}

	if commitMsg == nil {
		commitMsg = github.String("Update codeowners")
	}
	opt := &github.RepositoryContentFileOptions{
		Message: commitMsg,
		Content: []byte(newContent),
		SHA:     github.String(old.GetSHA()),
		Branch:  github.String(prBranch),
	}
	if _, _, err := cli.Repositories.CreateFile(ctx, owner, name, old.GetPath(), opt); err != nil {
		return errors.Wrap(err, "cli.Repositories.CreateFile")
	}
	return nil
}

func OpenPR(ctx context.Context, cli *github.Client, r *github.Repository, prTitle, head, body string, reviewReq *github.ReviewersRequest) (*github.PullRequest, error) {
	req := &github.NewPullRequest{
		Title: github.String(prTitle),
		Head:  github.String(head),
		Base:  r.DefaultBranch,
		Body:  github.String(body),
		Draft: github.Bool(false),
	}
	pr, _, err := cli.PullRequests.Create(ctx, r.GetOwner().GetLogin(), r.GetName(), req)
	if err != nil {
		return nil, errors.Wrap(err, "cli.PullRequests.Create")
	}

	if reviewReq != nil {
		if _, _, err := cli.PullRequests.RequestReviewers(ctx, r.GetOwner().GetLogin(), r.GetName(), pr.GetNumber(), *reviewReq); err != nil {
			return nil, errors.Wrap(err, "cli.PullRequests.RequestReviewers")
		}
	}
	return pr, nil
}

func isBranchExists(cli *github.Client, ctx context.Context, r *github.Repository, branch string) (bool, error) {
	_, res, err := cli.Repositories.GetBranch(ctx, r.GetOwner().GetLogin(), r.GetName(), branch)
	if err != nil {
		if res != nil && res.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, errors.Wrap(err, "cli.Repositories.GetBranch")
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
			return nil, errors.Wrap(ErrNotFound, "cli.Repositories.GetContents")
		}
		return nil, errors.Wrap(err, "cli.Repositories.GetContents")
	}

	return fc, nil
}
