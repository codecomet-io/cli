package testobs

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
)

func randSeq(n int) string {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

type CISystem string

const (
	GithubActions CISystem = "Github Actions"
	CircleCI               = "CircleCI"
	CINotDetected          = "(unknown)"
)

type CISystemVars struct {
	System          CISystem
	Repository      string
	RepositoryOwner string
	Branch          string
	CommitHash      string
	SeqBuildID      string
}

// findGitRepo searches for a .git directory starting from the current directory and moving up.
func findGitRepo(path string) (string, error) {
	for {
		_, err := os.Stat(filepath.Join(path, ".git"))
		if err == nil {
			return path, nil
		}
		if !os.IsNotExist(err) {
			// Some other error occurred
			return "", err
		}

		parent := filepath.Dir(path)
		if parent == path {
			// No .git directory found
			return "", fmt.Errorf(".git directory not found")
		}
		path = parent
	}
}

// extractRepoOwnerAndName attempts to parse the repository owner and name from the remote URL.
func extractRepoOwnerAndName(remoteUrl string) (string, string) {
	// Remove the protocol (https://, git://, ssh://, etc.) and optional user credentials
	if at := strings.Index(remoteUrl, "@"); at != -1 {
		remoteUrl = remoteUrl[at+1:]
	} else if start := strings.Index(remoteUrl, "://"); start != -1 {
		remoteUrl = remoteUrl[start+3:]
	}

	// Remove anything before a colon, which might be part of SSH URL or port number
	if colon := strings.LastIndex(remoteUrl, ":"); colon != -1 {
		remoteUrl = remoteUrl[colon+1:]
	}

	// Remove '.git' suffix and split by '/'
	remoteUrl = strings.TrimSuffix(remoteUrl, ".git")
	parts := strings.Split(remoteUrl, "/")
	if len(parts) >= 2 {
		owner := parts[len(parts)-2]
		repoName := parts[len(parts)-1]
		return owner, repoName
	}

	return "", ""
}

// getRepoDetails retrieves the owner and repo name from the remote URL, current branch, and latest commit hash.
func getRepoDetails() (string, string, string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", "", "", err
	}

	repoPath, err := findGitRepo(cwd)
	if err != nil {
		return "", "", "", "", err
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", "", "", "", err
	}

	remotes, err := repo.Remotes()
	if err != nil {
		return "", "", "", "", err
	}

	var owner, repoName string
	if len(remotes) > 0 {
		remoteUrl := remotes[0].Config().URLs[0]
		owner, repoName = extractRepoOwnerAndName(remoteUrl)
	} else {
		repoName = filepath.Base(repoPath)
	}

	head, err := repo.Head()
	if err != nil {
		return "", "", "", "", err
	}

	branch := strings.TrimPrefix(head.Name().String(), "refs/heads/")
	commit := head.Hash().String()

	return owner, repoName, branch, commit, nil
}

// Automatically detect some CI environments
func AutodetectCI() CISystemVars {
	vars := CISystemVars{}

	switch {
	case os.Getenv("GITHUB_ACTIONS") == "true":
		// https://docs.github.com/en/actions/learn-github-actions/variables
		vars.System = GithubActions
		repoWithOwner := os.Getenv("GITHUB_REPOSITORY")
		owner := os.Getenv("GITHUB_REPOSITORY_OWNER")
		vars.Repository = strings.TrimPrefix(repoWithOwner, owner+"/")
		vars.RepositoryOwner = owner
		vars.Branch = os.Getenv("GITHUB_REF_NAME")
		vars.CommitHash = os.Getenv("GITHUB_SHA")
		vars.SeqBuildID = "GHA" + ":" + os.Getenv("GITHUB_RUN_NUMBER") + ":" +
			os.Getenv("GITHUB_RUN_ATTEMPT")

	case os.Getenv("CIRCLECI") == "true":
		// https://circleci.com/docs/variables/
		vars.System = CircleCI
		vars.Branch = os.Getenv("CIRCLE_BRANCH")
		vars.Repository = os.Getenv("CIRCLE_PROJECT_REPONAME")
		vars.RepositoryOwner = os.Getenv("CIRCLE_PROJECT_USERNAME")
		vars.CommitHash = os.Getenv("CIRCLE_SHA1")
		vars.SeqBuildID = "CCI" + ":" + os.Getenv("CIRCLE_BUILD_NUM")
	default:
		return detectFromGit()
	}

	return vars
}

func detectFromGit() CISystemVars {
	vars := CISystemVars{}
	vars.System = CINotDetected

	repoOwner, repoName, branch, commit, err := getRepoDetails()
	if err != nil {
		fmt.Printf("Error detecting git repo: %v\nWill use environment variables.\n", err)

		// attempt to figure out variables from environment
		vars.Branch = os.Getenv("CODECOMET_BRANCH")
		vars.Repository = os.Getenv("CODECOMET_REPOSITORY")
		vars.RepositoryOwner = os.Getenv("CODECOMET_REPOSITORY_OWNER")
		vars.CommitHash = os.Getenv("CODECOMET_COMMIT_HASH")
	} else {
		vars.Branch = branch
		vars.Repository = repoName
		vars.RepositoryOwner = repoOwner
		vars.CommitHash = commit
	}

	vars.SeqBuildID = os.Getenv("CODECOMET_SEQ_BUILD_ID")
	if vars.SeqBuildID == "" {
		// last fall-back
		vars.SeqBuildID = time.Now().Format("060102-150405")
	}
	return vars
}
