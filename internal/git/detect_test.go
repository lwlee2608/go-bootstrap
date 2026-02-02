package git

import "testing"

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "SSH format with .git",
			input:  "origin\tgit@github.com:lwlee2608/go-bootstrap.git (fetch)\norigin\tgit@github.com:lwlee2608/go-bootstrap.git (push)",
			expect: "github.com/lwlee2608/go-bootstrap",
		},
		{
			name:   "SSH format without .git",
			input:  "origin\tgit@github.com:user/myrepo (fetch)",
			expect: "github.com/user/myrepo",
		},
		{
			name:   "HTTPS format with .git",
			input:  "origin\thttps://github.com/lwlee2608/go-bootstrap.git (fetch)",
			expect: "github.com/lwlee2608/go-bootstrap",
		},
		{
			name:   "HTTPS format without .git",
			input:  "origin\thttps://github.com/user/myrepo (fetch)",
			expect: "github.com/user/myrepo",
		},
		{
			name:   "HTTPS format with username",
			input:  "origin\thttps://lwlee2608@github.com/lwlee2608/go-bootstrap.git (fetch)",
			expect: "github.com/lwlee2608/go-bootstrap",
		},
		{
			name:   "GitLab SSH",
			input:  "origin\tgit@gitlab.com:company/project.git (fetch)",
			expect: "gitlab.com/company/project",
		},
		{
			name:   "No origin",
			input:  "upstream\tgit@github.com:other/repo.git (fetch)",
			expect: "",
		},
		{
			name:   "Empty output",
			input:  "",
			expect: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRemoteURL(tt.input)
			if got != tt.expect {
				t.Errorf("parseRemoteURL() = %q, want %q", got, tt.expect)
			}
		})
	}
}
