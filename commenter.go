package lint2hub

import (
	"context"
	"errors"

	"github.com/google/go-github/github"
)

var (
	ErrShaDoesNotMatch        = errors.New("latest pull request SHA does not match provided SHA")
	ErrFileNotFoundInDiff     = errors.New("file not found in diff")
	ErrPositionNotFoundInDiff = errors.New("position not found in diff")
)

type Comment struct {
	File string
	Line int
	Body string
}

type Commenter struct {
	gh          *github.Client
	owner       string
	repo        string
	pullRequest int
	sha         string

	filePositions    map[string]map[int]int
	existingComments []*github.PullRequestComment
}

func NewCommenter(gh *github.Client, owner string, repo string, pullRequest int, sha string) *Commenter {
	return &Commenter{
		gh:          gh,
		owner:       owner,
		repo:        repo,
		pullRequest: pullRequest,
		sha:         sha,
	}
}

func (c *Commenter) EnsureCommentPosted(ctx context.Context, comment *Comment) (*github.PullRequestComment, error) {
	if c.filePositions == nil {
		if err := c.hydrateFilePositions(ctx); err != nil {
			return nil, err
		}
	}
	if c.existingComments == nil {
		if err := c.hydrateExistingComments(ctx); err != nil {
			return nil, err
		}
	}

	filePositions, ok := c.filePositions[comment.File]
	if !ok {
		return nil, ErrFileNotFoundInDiff
	}
	filePosition, ok := filePositions[comment.Line]
	if !ok {
		return nil, ErrPositionNotFoundInDiff
	}

	ghComment := &github.PullRequestComment{
		Body:     github.String(comment.Body),
		Path:     github.String(comment.File),
		CommitID: github.String(c.sha),
		Position: github.Int(filePosition),
	}

	for _, existingComment := range c.existingComments {
		if existingComment.Body != nil && *ghComment.Body == *existingComment.Body &&
			existingComment.Path != nil && *ghComment.Path == *existingComment.Path &&
			existingComment.Position != nil && *ghComment.Position == *existingComment.Position {
			// Comment already exists
			return existingComment, nil
		}
	}

	ghComment, _, err := c.gh.PullRequests.CreateComment(ctx, c.owner, c.repo, c.pullRequest, ghComment)
	return ghComment, err
}

func (c *Commenter) hydrateFilePositions(ctx context.Context) error {
	// Grab the changes _before_ we compare SHA so there is no race between
	// grabbing the SHA and grabbing the diffs
	diff, _, err := c.gh.PullRequests.GetRaw(ctx, c.owner, c.repo, c.pullRequest, github.RawOptions{Type: github.Diff})
	if err != nil {
		return err
	}

	pr, _, err := c.gh.PullRequests.Get(ctx, c.owner, c.repo, c.pullRequest)
	if err != nil {
		return err
	}
	if pr.Head.SHA != nil && *pr.Head.SHA != c.sha {
		return ErrShaDoesNotMatch
	}

	c.filePositions = make(map[string]map[int]int)
	for file, fileDiff := range SplitDiffByFile(diff) {
		c.filePositions[file] = BuildPositionMap(fileDiff)
	}
	return nil
}

func (c *Commenter) hydrateExistingComments(ctx context.Context) error {
	opts := &github.PullRequestListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	c.existingComments = make([]*github.PullRequestComment, 0)
	for {
		existingComments, resp, err := c.gh.PullRequests.ListComments(ctx, c.owner, c.repo, c.pullRequest, opts)
		if err != nil {
			return err
		}

		c.existingComments = append(c.existingComments, existingComments...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return nil
}
