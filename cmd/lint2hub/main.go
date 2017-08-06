package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alindeman/lint2hub"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type urlFlag struct {
	URL *url.URL
}

func (f *urlFlag) Set(s string) error {
	url, err := url.Parse(s)
	if err != nil {
		return err
	}

	*f.URL = *url
	return nil
}

func (f *urlFlag) String() string {
	if f.URL == nil {
		return ""
	}
	return f.URL.String()
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}
}

func run() error {
	var (
		githubAccessToken string
		githubBaseURL     *url.URL
		owner             string
		repo              string
		pullRequest       int
		sha               string
		batch             bool
		file              string
		lineNum           int
		body              string
		timeout           time.Duration
	)

	flag.StringVar(&githubAccessToken, "github-access-token", "", "Access token for GitHub API")
	flag.Var(&urlFlag{URL: githubBaseURL}, "github-base-url", "Base URL for the GitHub API (defaults to the public GitHub API)")
	flag.StringVar(&owner, "owner", "", "Owner of the GitHub repository (i.e., the username or organization name)")
	flag.StringVar(&repo, "repo", "", "Name of the GitHub repository")
	flag.IntVar(&pullRequest, "pull-request", 0, "Pull request number")
	flag.StringVar(&sha, "sha", "", "SHA of the commit of this checkout. If this SHA does not match the latest SHA of the pull request, no comments will be posted")
	flag.BoolVar(&batch, "batch", false, "Batch mode")
	flag.StringVar(&file, "file", "", "Filename")
	flag.IntVar(&lineNum, "line", 0, "Line number")
	flag.StringVar(&body, "body", "", "Body of the comment")
	flag.DurationVar(&timeout, "timeout", 30*time.Second, "Timeout")

	flag.VisitAll(func(f *flag.Flag) {
		if value := os.Getenv(fmt.Sprintf("LINT2HUB_%s", strings.ToUpper(f.Name))); value != "" {
			_ = f.Value.Set(value)
		}
	})
	flag.Parse()

	if githubAccessToken == "" {
		return errors.New("required flag missing: -github-access-token")
	}
	if owner == "" {
		return errors.New("required flag missing: -owner")
	}
	if repo == "" {
		return errors.New("required flag missing: -repo")
	}
	if pullRequest == 0 {
		return errors.New("required flag missing: -pull-request")
	}
	if sha == "" {
		return errors.New("required flag missing: -sha")
	}
	if batch {
		if file != "" {
			return errors.New("both -file and -batch cannot be specified at the same time")
		}
		if lineNum != 0 {
			return errors.New("both -line and -batch cannot be specified at the same time")
		}
		if body != "" {
			return errors.New("both -body and -batch cannot be specified at the same time")
		}
	} else {
		if file == "" {
			return errors.New("required flag missing: -file")
		}
		if lineNum == 0 {
			return errors.New("required flag missing: -line")
		}
		if body == "" {
			return errors.New("required flag missing: -body")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubAccessToken})
	tc := oauth2.NewClient(ctx, ts)
	gh := github.NewClient(tc)
	if githubBaseURL != nil {
		gh.BaseURL = githubBaseURL
	}
	commenter, err := lint2hub.NewCommenter(ctx, gh, owner, repo, pullRequest, sha)
	if err != nil {
		return err
	}

	if batch {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			// filename<tab>line<tab>comment\n
			line := strings.TrimRight(scanner.Text(), "\r\n")
			parts := strings.SplitN(line, "\t", 3)
			if len(parts) < 3 {
				return fmt.Errorf("malformed batch line, must have 3 parts separated by tab characters: '%v'", line)
			}

			lineNum, err := strconv.Atoi(parts[1])
			if err != nil {
				return fmt.Errorf("cannot convert line number '%v' to integer: %v", parts[1], err)
			}
			file := parts[0]
			body := parts[2]

			if position, ok := commenter.GetPosition(file, lineNum); ok {
				if err := commenter.Post(ctx, file, position, body); err != nil {
					return err
				}
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}
	} else {
		if position, ok := commenter.GetPosition(file, lineNum); ok {
			if err := commenter.Post(ctx, file, position, body); err != nil {
				return err
			}
		}
	}
	return nil
}
