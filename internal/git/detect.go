package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// DetectAppName returns the current directory name as the suggested app name.
func DetectAppName() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Base(dir)
}

// DetectModuleName attempts to detect a Go module name from the git remote URL.
// It runs `git remote -v` and parses the origin URL.
// Returns empty string if detection fails.
func DetectModuleName() string {
	cmd := exec.Command("git", "remote", "-v")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return parseRemoteURL(string(output))
}

// parseRemoteURL extracts a Go module path from git remote -v output.
// Supports both SSH (git@github.com:user/repo.git) and HTTPS (https://github.com/user/repo.git) formats.
func parseRemoteURL(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "origin") {
			continue
		}

		// SSH format: git@github.com:user/repo.git
		sshPattern := regexp.MustCompile(`git@([^:]+):([^/]+)/([^/\s]+?)(?:\.git)?(?:\s|$)`)
		if matches := sshPattern.FindStringSubmatch(line); len(matches) == 4 {
			host := matches[1]
			user := matches[2]
			repo := strings.TrimSuffix(matches[3], ".git")
			return host + "/" + user + "/" + repo
		}

		// HTTPS format: https://github.com/user/repo.git or https://user@github.com/user/repo.git
		httpsPattern := regexp.MustCompile(`https?://(?:[^@]+@)?([^/]+)/([^/]+)/([^/\s]+?)(?:\.git)?(?:\s|$)`)
		if matches := httpsPattern.FindStringSubmatch(line); len(matches) == 4 {
			host := matches[1]
			user := matches[2]
			repo := strings.TrimSuffix(matches[3], ".git")
			return host + "/" + user + "/" + repo
		}
	}

	return ""
}
