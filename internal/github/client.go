package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var (
	ErrNotFound         = errors.New("github resource not found")
	ErrRateLimited      = errors.New("github rate limit exceeded")
	ErrNoReleases       = errors.New("repository has no releases")
	ErrUnexpectedStatus = errors.New("unexpected github status")
)

type Client interface {
	RepositoryExists(ctx context.Context, owner, repo string) (bool, error)
	GetLatestRelease(ctx context.Context, owner, repo string) (string, string, error)
}

type client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

type repositoryResponse struct {
	FullName string `json:"full_name"`
}

type latestReleaseResponse struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

func NewClient(token string) Client {
	return &client{
		baseURL: "https://api.github.com",
		token:   token,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *client) RepositoryExists(ctx context.Context, owner, repo string) (bool, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s", c.baseURL, owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	case http.StatusTooManyRequests, http.StatusForbidden:
		if isRateLimited(resp) {
			return false, ErrRateLimited
		}
		return false, fmt.Errorf("%w: %d", ErrUnexpectedStatus, resp.StatusCode)
	default:
		return false, fmt.Errorf("%w: %d", ErrUnexpectedStatus, resp.StatusCode)
	}
}

func (c *client) GetLatestRelease(ctx context.Context, owner, repo string) (string, string, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s/releases/latest", c.baseURL, owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", "", err
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var release latestReleaseResponse
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return "", "", err
		}
		return release.TagName, release.HTMLURL, nil
	case http.StatusNotFound:
		return "", "", ErrNoReleases
	case http.StatusTooManyRequests, http.StatusForbidden:
		if isRateLimited(resp) {
			return "", "", ErrRateLimited
		}
		return "", "", fmt.Errorf("%w: %d", ErrUnexpectedStatus, resp.StatusCode)
	default:
		return "", "", fmt.Errorf("%w: %d", ErrUnexpectedStatus, resp.StatusCode)
	}
}

func (c *client) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	if strings.TrimSpace(c.token) != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

func isRateLimited(resp *http.Response) bool {
	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}

	remaining := resp.Header.Get("X-RateLimit-Remaining")
	if resp.StatusCode == http.StatusForbidden && remaining == "0" {
		return true
	}

	return false
}
