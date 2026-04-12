package stack

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComposerize_Basic(t *testing.T) {
	yaml, err := Composerize("docker run -d --name myapp -p 8080:80 nginx:latest")
	require.NoError(t, err)
	assert.Contains(t, yaml, "image: nginx:latest")
	assert.Contains(t, yaml, "8080:80")
	assert.Contains(t, yaml, "myapp:")
}

func TestComposerize_WithEnv(t *testing.T) {
	yaml, err := Composerize("docker run -e FOO=bar -e BAZ=qux alpine")
	require.NoError(t, err)
	assert.Contains(t, yaml, "FOO=bar")
	assert.Contains(t, yaml, "BAZ=qux")
}

func TestComposerize_WithVolumes(t *testing.T) {
	yaml, err := Composerize("docker run -v /host/path:/container/path nginx")
	require.NoError(t, err)
	assert.Contains(t, yaml, "/host/path:/container/path")
}

func TestComposerize_WithRestart(t *testing.T) {
	yaml, err := Composerize("docker run --restart unless-stopped nginx")
	require.NoError(t, err)
	assert.Contains(t, yaml, "restart: unless-stopped")
}

func TestComposerize_WithCommand(t *testing.T) {
	yaml, err := Composerize("docker run alpine echo hello world")
	require.NoError(t, err)
	assert.Contains(t, yaml, "image: alpine")
	// command: echo or command: [echo, hello, world]
	assert.True(t, strings.Contains(yaml, "echo") && strings.Contains(yaml, "hello"))
}

func TestComposerize_NoImage(t *testing.T) {
	_, err := Composerize("docker run -d -p 80:80")
	assert.Error(t, err)
}

func TestComposerize_Privileged(t *testing.T) {
	yaml, err := Composerize("docker run --privileged alpine")
	require.NoError(t, err)
	assert.Contains(t, yaml, "privileged: true")
}

func TestComposerize_StripPrefix(t *testing.T) {
	y1, err1 := Composerize("nginx")
	require.NoError(t, err1)
	y2, err2 := Composerize("docker run nginx")
	require.NoError(t, err2)
	y3, err3 := Composerize("docker container run nginx")
	require.NoError(t, err3)
	assert.Equal(t, y1, y2)
	assert.Equal(t, y2, y3)
}

func TestComposerize_QuotedArgs(t *testing.T) {
	yaml, err := Composerize(`docker run -e "MY_VAR=hello world" alpine`)
	require.NoError(t, err)
	assert.Contains(t, yaml, "MY_VAR=hello world")
}

func TestComposerize_ParseMountToVolume(t *testing.T) {
	yaml, err := Composerize("docker run --mount type=bind,source=/src,target=/dst alpine")
	require.NoError(t, err)
	assert.Contains(t, yaml, "/src:/dst")
}

func TestComposerize_UnknownShortFlagAsCommand(t *testing.T) {
	// -s is not a docker run flag — it should be passed as the container command,
	// not silently dropped. This is the iperf3 server mode flag.
	yaml, err := Composerize("docker run -d --name iperf3-server -p 5201:5201 networkstatic/iperf3 -s")
	require.NoError(t, err)
	assert.Contains(t, yaml, "image: networkstatic/iperf3")
	assert.Contains(t, yaml, "command: -s")
}

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"a b c", []string{"a", "b", "c"}},
		{`a "b c" d`, []string{"a", "b c", "d"}},
		{`a 'b c' d`, []string{"a", "b c", "d"}},
		{`a "b \"c\"" d`, []string{"a", `b "c"`, "d"}},
	}
	for _, tc := range tests {
		got, err := splitArgs(tc.input)
		assert.NoError(t, err)
		assert.Equal(t, tc.expected, got, "input: %q", tc.input)
	}
}

func TestParseImageRef(t *testing.T) {
	tests := []struct {
		image    string
		registry string
		repo     string
		tag      string
	}{
		{"nginx", "registry-1.docker.io", "library/nginx", "latest"},
		{"nginx:1.25", "registry-1.docker.io", "library/nginx", "1.25"},
		{"myorg/myapp:v2", "registry-1.docker.io", "myorg/myapp", "v2"},
		{"ghcr.io/foo/bar:latest", "ghcr.io", "foo/bar", "latest"},
		{"registry.example.com:5000/myapp:dev", "registry.example.com:5000", "myapp", "dev"},
	}
	for _, tc := range tests {
		reg, repo, tag := ParseImageRef(tc.image)
		assert.Equal(t, tc.registry, reg, "registry for %q", tc.image)
		assert.Equal(t, tc.repo, repo, "repo for %q", tc.image)
		assert.Equal(t, tc.tag, tag, "tag for %q", tc.image)
	}
}
