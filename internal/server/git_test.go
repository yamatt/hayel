package server

import "testing"

func TestRepositoryPath(t *testing.T) {
	tests := []struct {
		name     string
		request  string
		repo     string
		endpoint bool
		valid    bool
	}{
		// Valid v1 single-level paths
		{name: "single level repository", request: "/repo-name", repo: "repo-name", endpoint: false, valid: true},
		{name: "single level with .git", request: "/repo-name.git", repo: "repo-name.git", endpoint: false, valid: true},
		{name: "info refs endpoint", request: "/repo-name/info/refs", repo: "repo-name", endpoint: true, valid: true},
		{name: "receive pack endpoint", request: "/repo-name/git-receive-pack", repo: "repo-name", endpoint: true, valid: true},
		{name: "upload pack endpoint", request: "/repo-name/git-upload-pack", repo: "repo-name", endpoint: true, valid: true},

		// Multi-level paths disallowed in v1
		{name: "group nested path disallowed", request: "/group/repo-name", valid: false},
		{name: "group info refs disallowed", request: "/group/repo-name/info/refs", valid: false},
		{name: "deeply nested group disallowed", request: "/team/project/repo-name/info/refs", valid: false},

		// Invalid requests
		{name: "root path", request: "/", valid: false},
		{name: "parent traversal", request: "/../repo-name", valid: false},
		{name: "dot component", request: "/./repo-name", valid: false},
		{name: "backslash", request: "/repo\\name", valid: false},
		{name: "double slash", request: "//repo-name", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, endpoint, valid := repositoryPath(tt.request)
			if repo != tt.repo || endpoint != tt.endpoint || valid != tt.valid {
				t.Fatalf("repositoryPath(%q) = (%q, %t, %t), want (%q, %t, %t)", tt.request, repo, endpoint, valid, tt.repo, tt.endpoint, tt.valid)
			}
		})
	}
}
