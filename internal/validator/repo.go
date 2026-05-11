package validator

import (
	"errors"
	"strings"
)

func ValidateRepo(repo string) error {
	if repo == "" {
		return errors.New("repo is required")
	}

	parts := strings.Split(repo, "/")
	const githubRepoPartsCount = 2

	if len(parts) != githubRepoPartsCount {
		return errors.New("repo must be in format owner/repo")
	}

	if parts[0] == "" || parts[1] == "" {
		return errors.New("repo must be in format owner/repo")
	}

	return nil
}

// ParseRepo splits a validated "owner/repo" string into its components.
// Must only be called after ValidateRepo succeeds.
func ParseRepo(repo string) (owner, name string) {
	parts := strings.Split(repo, "/")
	return parts[0], parts[1]
}
