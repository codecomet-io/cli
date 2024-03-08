package testobs

import (
	"testing"

	"github.com/matryer/is"
)

func TestDetectFromGit(t *testing.T) {
	is := is.New(t)

	vars := detectFromGit()
	is.Equal(vars.Repository, "cli")
	is.Equal(vars.RepositoryOwner, "codecomet-io")
}

// TestExtractRepoOwnerAndName is a table-driven test for the extractRepoOwnerAndName function.
func TestExtractRepoOwnerAndName(t *testing.T) {
	tests := []struct {
		url           string
		expectedOwner string
		expectedName  string
	}{
		{"https://<token>:x-oauth-basic@github.com/codecomet-io/cli", "codecomet-io", "cli"},
		{"https://username:password@github.com/codecomet-io/cli", "codecomet-io", "cli"},
		{"https://github.com:8443/codecomet-io/cli", "codecomet-io", "cli"},
		{"git@github.com:codecomet-io/cli", "codecomet-io", "cli"},
		{"ssh://git@github.com:2222/codecomet-io/cli", "codecomet-io", "cli"},
		{"git://github.com/codecomet-io/cli", "codecomet-io", "cli"},
		{"customuser@github.com:codecomet-io/cli", "codecomet-io", "cli"},
		{"https://github.com/codecomet-io/cli", "codecomet-io", "cli"},
		{"ssh://git@192.168.1.100:9876/codecomet-io/cli", "codecomet-io", "cli"},
		{"git://github.com:9418/codecomet-io/cli", "codecomet-io", "cli"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			is := is.New(t)
			owner, name := extractRepoOwnerAndName(tt.url)
			is.Equal(owner, tt.expectedOwner)
			is.Equal(name, tt.expectedName)
		})
	}
}
