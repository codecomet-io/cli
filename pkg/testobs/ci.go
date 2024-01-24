package testobs

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
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
		vars.System = CINotDetected
		// attempt to figure out variables from environment
		vars.Branch = os.Getenv("CODECOMET_BRANCH")
		vars.Repository = os.Getenv("CODECOMET_REPOSITORY")
		vars.RepositoryOwner = os.Getenv("CODECOMET_REPOSITORY_OWNER")
		vars.CommitHash = os.Getenv("CODECOMET_COMMIT_HASH")
		vars.SeqBuildID = os.Getenv("CODECOMET_SEQ_BUILD_ID")
		if vars.SeqBuildID == "" {
			// last fall-back
			vars.SeqBuildID = randSeq(8)
		}
	}

	return vars
}
