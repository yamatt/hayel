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
		{name: "repository", request: "/group/repo-name", repo: "group/repo-name", valid: true},
		{name: "info refs", request: "/group/repo-name/info/refs", repo: "group/repo-name", endpoint: true, valid: true},
		{name: "receive pack", request: "/group/repo-name/git-receive-pack", repo: "group/repo-name", endpoint: true, valid: true},
		{name: "nested group", request: "/team/project/repo-name/info/refs", repo: "team/project/repo-name", endpoint: true, valid: true},
		{name: "missing repository", request: "/group", valid: false},
		{name: "parent traversal", request: "/group/../repo-name", valid: false},
		{name: "backslash", request: "/group\\repo-name", valid: false},
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
