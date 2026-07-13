package cmd

import "testing"

func TestMaskToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		token string
		want  string
	}{
		{
			name:  "long token keeps first and last four characters",
			token: "secret_abcdefghijklmnop",
			want:  "secr****mnop",
		},
		{
			name:  "nine characters is the shortest masked form",
			token: "123456789",
			want:  "1234****6789",
		},
		{
			name:  "eight characters is fully masked",
			token: "12345678",
			want:  "****",
		},
		{
			name:  "short token is fully masked",
			token: "abc",
			want:  "****",
		},
		{
			name:  "empty token",
			token: "",
			want:  "****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := maskToken(tt.token); got != tt.want {
				t.Errorf("maskToken(%q) = %q, want %q", tt.token, got, tt.want)
			}
		})
	}
}
