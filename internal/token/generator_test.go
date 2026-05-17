package token_test

import (
	"testing"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/token"
)

func TestGenerator_Generate(t *testing.T) {
	cases := []struct {
		name string
	}{
		{name: "returns non-empty string without error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := token.New()
			tok, err := g.Generate()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tok == "" {
				t.Fatal("expected non-empty token")
			}
		})
	}
}
