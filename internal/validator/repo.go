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
	if len(parts) != 2 {
		return errors.New("repo must be in format owner/repo")
	}

	if parts[0] == "" || parts[1] == "" {
		return errors.New("repo must be in format owner/repo")
	}

	return nil
}
