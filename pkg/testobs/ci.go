package testobs

import "os"

type CISystem string

const (
	GithubActions CISystem = "Github Actions"
	CircleCI               = "CircleCI"
	CINotDetected          = "(none)"
)

// Automatically detect some CI environments
func AutodetectCI() CISystem {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		// https://docs.github.com/en/actions/learn-github-actions/variables
		return GithubActions
	}
	if os.Getenv("CIRCLECI") == "true" {
		// https://circleci.com/docs/variables/
		return CircleCI
	}

	return CINotDetected
}
