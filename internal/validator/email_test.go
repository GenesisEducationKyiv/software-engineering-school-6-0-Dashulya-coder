package validator

import "testing"

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "valid email",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: true,
		},
		{
			name:    "invalid email without at",
			email:   "testexample.com",
			wantErr: true,
		},
		{
			name:    "invalid email without domain",
			email:   "test@",
			wantErr: true,
		},
		{
			name:    "invalid email without top level domain",
			email:   "test@example",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
