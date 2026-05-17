package urlbuilder_test

import (
	"testing"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/urlbuilder"
)

func TestBuilder_ConfirmURL(t *testing.T) {
	cases := []struct {
		name     string
		baseURL  string
		token    string
		expected string
	}{
		{
			name:     "constructs confirm URL",
			baseURL:  "http://localhost:8080",
			token:    "abc123",
			expected: "http://localhost:8080/api/confirm/abc123",
		},
		{
			name:     "constructs confirm URL with different base",
			baseURL:  "https://example.com",
			token:    "xyz789",
			expected: "https://example.com/api/confirm/xyz789",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := urlbuilder.New(tc.baseURL)
			if got := b.ConfirmURL(tc.token); got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestBuilder_UnsubscribeURL(t *testing.T) {
	cases := []struct {
		name     string
		baseURL  string
		token    string
		expected string
	}{
		{
			name:     "constructs unsubscribe URL",
			baseURL:  "http://localhost:8080",
			token:    "abc123",
			expected: "http://localhost:8080/api/unsubscribe/abc123",
		},
		{
			name:     "constructs unsubscribe URL with different base",
			baseURL:  "https://example.com",
			token:    "xyz789",
			expected: "https://example.com/api/unsubscribe/xyz789",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := urlbuilder.New(tc.baseURL)
			if got := b.UnsubscribeURL(tc.token); got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}
