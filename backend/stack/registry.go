package stack

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
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

	// Normalize Docker Hub aliases to the canonical registry endpoint.
	// Docker Engine treats "docker.io" and "index.docker.io" as aliases for
	// "registry-1.docker.io"; we must do the same so the registry URL we
	// build actually points at the OCI distribution API rather than the
	// docker.com marketing website (which would return an HTML page and
	// trigger a false-positive update detection).
	if registry == "docker.io" || registry == "index.docker.io" {
		registry = "registry-1.docker.io"
	}

	// Docker Hub official images: "nginx" → "library/nginx".
	if registry == "registry-1.docker.io" && !strings.Contains(repository, "/") {
		repository = "library/" + repository
	}
	return
}

// manifestAccept is the Accept header value sent for all manifest requests.
var manifestAccept = strings.Join([]string{
	"application/vnd.oci.image.index.v1+json",
	"application/vnd.docker.distribution.manifest.list.v2+json",
	"application/vnd.oci.image.manifest.v1+json",
	"application/vnd.docker.distribution.manifest.v2+json",
}, ", ")

// GetRegistryDigest fetches the current manifest digest for the given image
// tag from its registry, using the OCI Distribution HTTP API.
//
// It uses the standard WWW-Authenticate challenge flow to obtain a Bearer
// token, which works with any OCI-compliant registry (Docker Hub, GHCR,
// lscr.io, Quay, Harbor, etc.). If HEAD succeeds but the
// docker-content-digest header is absent, it falls back to GET and computes
// the digest from the response body.
func GetRegistryDigest(registry, repository, tag string) (string, error) {
	registryURL := registry
	if !strings.Contains(registryURL, "://") {
		registryURL = "https://" + registryURL
	}
	manifestURL := fmt.Sprintf("%s/v2/%s/manifests/%s", registryURL, repository, tag)

	// Resolve a Bearer token — try well-known endpoints first (Docker Hub,
	// GHCR), then fall back to the standard WWW-Authenticate challenge.
	token, _ := getAuthToken(registry, repository)
	if token == "" {
		token, _ = authViaChallenge(manifestURL)
	}

	// HEAD request.
	digest, err := fetchDigestHEAD(manifestURL, token)
	if err != nil {
		log.Debug().Err(err).Str("image", repository).Msg("HEAD manifest failed, falling back to GET")
	}
	if digest != "" {
		return digest, nil
	}

	// GET fallback — compute digest from body.
	digest, err = fetchDigestGET(manifestURL, token)
	if err != nil {
		return "", fmt.Errorf("registry GET: %w", err)
	}
	return digest, nil
}

func fetchDigestHEAD(manifestURL, token string) (string, error) {
	req, err := http.NewRequest(http.MethodHead, manifestURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", manifestAccept)
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
		return "", fmt.Errorf("HEAD returned status %d", resp.StatusCode)
	}
	return resp.Header.Get("docker-content-digest"), nil
}

func fetchDigestGET(manifestURL, token string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, manifestURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", manifestAccept)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("GET returned status %d", resp.StatusCode)
	}
	if d := resp.Header.Get("docker-content-digest"); d != "" {
		io.Copy(io.Discard, resp.Body)
		return d, nil
	}
	// Guard against being redirected to a non-registry site (e.g. a marketing
	// page). An HTML response body can never be a valid manifest, so hashing
	// it would produce a fake digest that causes a persistent false-positive
	// update notification.
	if ct := resp.Header.Get("Content-Type"); strings.HasPrefix(ct, "text/html") {
		io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("unexpected HTML response from registry (possible misconfigured registry URL)")
	}
	// Compute digest from body.
	h := sha256.New()
	if _, err := io.Copy(h, resp.Body); err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}

// getAuthToken returns a Bearer token using well-known endpoints for Docker
// Hub and GHCR. For any other registry it returns ("", nil) so the caller
// can fall back to the WWW-Authenticate challenge flow.
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
		return "", nil
	}
	return fetchBearerToken(authURL)
}

// authViaChallenge performs an unauthenticated HEAD against manifestURL,
// parses the WWW-Authenticate header on a 401 response, and fetches a
// Bearer token from the advertised realm. This is the standard OCI token
// flow and works with any compliant registry (lscr.io, quay.io, etc.).
func authViaChallenge(manifestURL string) (string, error) {
	req, err := http.NewRequest(http.MethodHead, manifestURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", manifestAccept)
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusUnauthorized {
		return "", nil // no auth required (or unsupported)
	}

	wwwAuth := resp.Header.Get("WWW-Authenticate")
	if !strings.HasPrefix(strings.ToLower(wwwAuth), "bearer ") {
		return "", fmt.Errorf("unsupported WWW-Authenticate: %s", wwwAuth)
	}

	params := parseBearerParams(wwwAuth)
	realm := params["realm"]
	if realm == "" {
		return "", fmt.Errorf("no realm in WWW-Authenticate")
	}

	tokenURL := realm
	var qparts []string
	if svc := params["service"]; svc != "" {
		qparts = append(qparts, "service="+svc)
	}
	if scope := params["scope"]; scope != "" {
		qparts = append(qparts, "scope="+scope)
	}
	if len(qparts) > 0 {
		sep := "?"
		if strings.Contains(tokenURL, "?") {
			sep = "&"
		}
		tokenURL += sep + strings.Join(qparts, "&")
	}

	log.Debug().Str("tokenURL", tokenURL).Msg("registry auth via challenge")
	return fetchBearerToken(tokenURL)
}

// parseBearerParams extracts key="value" pairs from a
// Bearer realm="...",service="...",scope="..." header.
func parseBearerParams(header string) map[string]string {
	out := make(map[string]string)
	idx := strings.Index(header, " ")
	if idx < 0 {
		return out
	}
	for _, part := range strings.Split(header[idx+1:], ",") {
		part = strings.TrimSpace(part)
		eq := strings.Index(part, "=")
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(part[:eq])
		val := strings.Trim(strings.TrimSpace(part[eq+1:]), `"`)
		out[key] = val
	}
	return out
}

// fetchBearerToken GETs the given token URL and returns the token.
func fetchBearerToken(authURL string) (string, error) {
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
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse auth token: %w", err)
	}
	// Some registries use "access_token" instead of "token".
	if result.Token != "" {
		return result.Token, nil
	}
	return result.AccessToken, nil
}
