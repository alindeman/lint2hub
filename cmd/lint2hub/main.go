package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alindeman/lint2hub"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}
}

func run() error {
	var (
		githubAccessToken string
		owner             string
		repo              string
		pullRequest       int
		sha               string
		batch             bool
		file              string
		line              int
		body              string
		timeout           time.Duration
	)

	flag.StringVar(&githubAccessToken, "github-access-token", "", "Access token for GitHub API")
	flag.StringVar(&owner, "owner", "", "Owner of the GitHub repository (i.e., the username or organization name)")
	flag.StringVar(&repo, "repo", "", "Name of the GitHub repository")
	flag.IntVar(&pullRequest, "pull-request", 0, "Pull request number")
	flag.StringVar(&sha, "sha", "", "SHA of the commit of this checkout. If this SHA does not match the latest SHA of the pull request, no comments will be posted")
	flag.BoolVar(&batch, "batch", false, "Batch mode")
	flag.StringVar(&file, "file", "", "Filename")
	flag.IntVar(&line, "line", 0, "Line number")
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
		if line != 0 {
			return errors.New("both -line and -batch cannot be specified at the same time")
		}
		if body != "" {
			return errors.New("both -body and -batch cannot be specified at the same time")
		}
	} else {
		if file == "" {
			return errors.New("required flag missing: -file")
		}
		if line == 0 {
			return errors.New("required flag missing: -line")
		}
		if body == "" {
			return errors.New("required flag missing: -body")
		}
	}

	log := log.New(os.Stderr, "", 0)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubAccessToken})
	tc := oauth2.NewClient(ctx, ts)
	gh := github.NewClient(tc)
	commenter := lint2hub.NewCommenter(gh, owner, repo, pullRequest, sha)

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

			comment := &lint2hub.Comment{
				File: parts[0],
				Line: lineNum,
				Body: parts[2],
			}
			if _, err := commenter.EnsureCommentPosted(ctx, comment); err != nil {
				if err = logMinorError(log, err); err != nil {
					return err
				}
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}

		return nil
	} else {
		comment := &lint2hub.Comment{
			File: file,
			Line: line,
			Body: body,
		}
		_, err := commenter.EnsureCommentPosted(ctx, comment)
		if err = logMinorError(log, err); err != nil {
			return err
		}
	}
	return nil
}

// logMinorError logs an error if it's recognized as a minor issue; if it's not
// recognized as a minor error, it's returned
func logMinorError(log *log.Logger, err error) error {
	if err == lint2hub.ErrShaDoesNotMatch {
		log.Printf("%v: comments will not be posted", err)
		return nil
	} else if err == lint2hub.ErrFileNotFoundInDiff {
		log.Printf("%v: comment will not be posted", err)
		return nil
	} else if err == lint2hub.ErrPositionNotFoundInDiff {
		log.Printf("%v: comment will not be posted", err)
		return nil
	}
	return err
}
