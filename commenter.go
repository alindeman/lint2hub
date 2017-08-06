package lint2hub

import (
	"context"
	"errors"

	"github.com/google/go-github/github"
)

var (
	// ErrSHANotLatest is returned if the latest pull request SHA does not
	// match the provided SHA
	ErrSHANotLatest = errors.New("latest pull request SHA does not match provided SHA")
)

// Commenter is a high level GitHub client capable of commenting on pull request
// diffs. It caches the diff at a particular SHA and also attempts to prevent
// duplicate line comments from being posted.
type Commenter struct {
	gh       *github.Client
	owner    string
	repo     string
	prNumber int
	sha      string

	diff     *diff
	comments []*github.PullRequestComment
}

// NewCommenter creates a new commenting client. If the provided sha is not the
// latest SHA, ErrSHANotLatest is returned to avoid commenting on an outdated
// diff.
func NewCommenter(ctx context.Context, gh *github.Client, owner string, repo string, prNumber int, sha string) (*Commenter, error) {
	// Grab the diff _before_ we compare SHA so there is no race between
	// grabbing the SHA and grabbing the diffs
	diffStr, _, err := gh.PullRequests.GetRaw(ctx, owner, repo, prNumber, github.RawOptions{Type: github.Diff})
	if err != nil {
		return nil, err
	}

	pr, _, err := gh.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, err
	}
	if pr.Head.SHA != nil && *pr.Head.SHA != sha {
		return nil, ErrSHANotLatest
	}

	comments, err := loadComments(ctx, gh, owner, repo, prNumber)
	if err != nil {
		return nil, err
	}

	return &Commenter{
		gh:       gh,
		owner:    owner,
		repo:     repo,
		prNumber: prNumber,
		sha:      sha,

		diff:     newDiff(diffStr),
		comments: comments,
	}, nil
}

// GetPosition retrieves the diff position for a given file and lineNum. Returns
// (0, false) if the position is not present in the diff, meaning a comment
// cannot be posted on that lineNum within the pull request.
func (c *Commenter) GetPosition(file string, lineNum int) (position int, ok bool) {
	return c.diff.GetPosition(file, lineNum)
}

// Post posts a comment at the given file and position. If a comment with the
// same body is already present at that position, Post does nothing.
func (c *Commenter) Post(ctx context.Context, file string, position int, body string) error {
	newComment := &github.PullRequestComment{
		Body:     github.String(body),
		Path:     github.String(file),
		CommitID: github.String(c.sha),
		Position: github.Int(position),
	}

	for _, comment := range c.comments {
		if comment.Body != nil && *newComment.Body == *comment.Body &&
			comment.Path != nil && *newComment.Path == *comment.Path &&
			comment.Position != nil && *newComment.Position == *comment.Position {
			// Comment already exists
			return nil
		}
	}

	newComment, _, err := c.gh.PullRequests.CreateComment(ctx, c.owner, c.repo, c.prNumber, newComment)
	c.comments = append(c.comments, newComment)

	return err
}

func loadComments(ctx context.Context, gh *github.Client, owner string, repo string, prNumber int) ([]*github.PullRequestComment, error) {
	opts := &github.PullRequestListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	allComments := make([]*github.PullRequestComment, 0)
	for {
		comments, resp, err := gh.PullRequests.ListComments(ctx, owner, repo, prNumber, opts)
		if err != nil {
			return nil, err
		}

		allComments = append(allComments, comments...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return allComments, nil
}
