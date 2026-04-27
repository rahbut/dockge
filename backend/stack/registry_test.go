package stack

import "testing"

func TestParseImageRef_LSCR(t *testing.T) {
	tests := []struct {
		image    string
		wantReg  string
		wantRepo string
		wantTag  string
	}{
		{"lscr.io/linuxserver/sonarr:latest", "lscr.io", "linuxserver/sonarr", "latest"},
		{"lscr.io/linuxserver/radarr:5.3.6", "lscr.io", "linuxserver/radarr", "5.3.6"},
		{"lscr.io/linuxserver/plex", "lscr.io", "linuxserver/plex", "latest"},
		{"ghcr.io/rahbut/dockge:latest", "ghcr.io", "rahbut/dockge", "latest"},
		{"nginx:latest", "registry-1.docker.io", "library/nginx", "latest"},
		{"nginx", "registry-1.docker.io", "library/nginx", "latest"},
		{"linuxserver/heimdall:latest", "registry-1.docker.io", "linuxserver/heimdall", "latest"},
		// Docker Hub aliases — must be normalised to registry-1.docker.io so the
		// manifest URL is built correctly (docker.io redirects to www.docker.com).
		{"docker.io/searxng/searxng:latest", "registry-1.docker.io", "searxng/searxng", "latest"},
		{"docker.io/searxng/searxng", "registry-1.docker.io", "searxng/searxng", "latest"},
		{"docker.io/library/nginx:latest", "registry-1.docker.io", "library/nginx", "latest"},
		{"index.docker.io/searxng/searxng:latest", "registry-1.docker.io", "searxng/searxng", "latest"},
	}
	for _, tt := range tests {
		reg, repo, tag := ParseImageRef(tt.image)
		if reg != tt.wantReg || repo != tt.wantRepo || tag != tt.wantTag {
			t.Errorf("ParseImageRef(%q) = (%q, %q, %q), want (%q, %q, %q)",
				tt.image, reg, repo, tag, tt.wantReg, tt.wantRepo, tt.wantTag)
		}
	}
}

func TestParseBearerParams(t *testing.T) {
	tests := []struct {
		header string
		want   map[string]string
	}{
		{
			`Bearer realm="https://ghcr.io/token",service="ghcr.io",scope="repository:linuxserver/sonarr:pull"`,
			map[string]string{
				"realm":   "https://ghcr.io/token",
				"service": "ghcr.io",
				"scope":   "repository:linuxserver/sonarr:pull",
			},
		},
		{
			`Bearer realm="https://auth.docker.io/token",service="registry.docker.io"`,
			map[string]string{
				"realm":   "https://auth.docker.io/token",
				"service": "registry.docker.io",
			},
		},
	}
	for _, tt := range tests {
		got := parseBearerParams(tt.header)
		for k, v := range tt.want {
			if got[k] != v {
				t.Errorf("parseBearerParams(%q)[%q] = %q, want %q", tt.header, k, got[k], v)
			}
		}
	}
}
