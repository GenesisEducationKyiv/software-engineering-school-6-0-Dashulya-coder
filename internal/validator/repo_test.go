package validator

import "testing"

func TestValidateRepo(t *testing.T) {
	tests := []struct {
		name    string
		repo    string
		wantErr bool
	}{
		{
			name:    "valid repo",
			repo:    "golang/go",
			wantErr: false,
		},
		{
			name:    "empty repo",
			repo:    "",
			wantErr: true,
		},
		{
			name:    "missing slash",
			repo:    "golanggo",
			wantErr: true,
		},
		{
			name:    "empty owner",
			repo:    "/go",
			wantErr: true,
		},
		{
			name:    "empty repo name",
			repo:    "golang/",
			wantErr: true,
		},
		{
			name:    "too many slashes",
			repo:    "a/b/c",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepo(tt.repo)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateRepo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
