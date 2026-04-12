package stack

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// ParseImageRef splits an image reference into registry, repository, and tag.
// Mirrors the TS Stack.parseImageRef() implementation exactly.
func ParseImageRef(image string) (registry, repository, tag string) {
	registry = "registry-1.docker.io"
	repository = image
	tag = "latest"

	// Split off tag (last colon that comes after the last slash).
	colonIdx := strings.LastIndex(repository, ":")
	slashIdx := strings.LastIndex(repository, "/")
	if colonIdx > slashIdx {
		tag = repository[colonIdx+1:]
		repository = repository[:colonIdx]
	}

	// Detect custom registry (prefix before first slash contains '.' or ':').
	firstSlash := strings.Index(repository, "/")
	if firstSlash > 0 {
		prefix := repository[:firstSlash]
		if strings.Contains(prefix, ".") || strings.Contains(prefix, ":") {
			registry = prefix
			repository = repository[firstSlash+1:]
		}
	}

	// Docker Hub official images: "nginx" → "library/nginx".
	if registry == "registry-1.docker.io" && !strings.Contains(repository, "/") {
		repository = "library/" + repository
	}
	return
}

// GetRegistryDigest fetches the current manifest digest for the given image
// tag from its registry, using the OCI Distribution HTTP API (HEAD request).
func GetRegistryDigest(registry, repository, tag string) (string, error) {
	token, err := getAuthToken(registry, repository)
	if err != nil {
		return "", fmt.Errorf("registry auth: %w", err)
	}

	registryURL := registry
	if !strings.Contains(registryURL, "://") {
		registryURL = "https://" + registryURL
	}
	manifestURL := fmt.Sprintf("%s/v2/%s/manifests/%s", registryURL, repository, tag)

	req, err := http.NewRequest(http.MethodHead, manifestURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.oci.image.index.v1+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.docker.distribution.manifest.v2+json",
	}, ", "))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	digest := resp.Header.Get("docker-content-digest")
	return digest, nil
}

func getAuthToken(registry, repository string) (string, error) {
	var authURL string
	switch registry {
	case "registry-1.docker.io":
		authURL = fmt.Sprintf(
			"https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull",
			repository)
	case "ghcr.io":
		authURL = fmt.Sprintf(
			"https://ghcr.io/token?scope=repository:%s:pull",
			repository)
	default:
		// Custom registries: attempt anonymous token endpoint; fail silently.
		return "", nil
	}

	resp, err := httpClient.Get(authURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse auth token: %w", err)
	}
	return result.Token, nil
}
