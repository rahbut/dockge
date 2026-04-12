// Package stack contains the Stack domain model and related helpers.
package stack

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// composeService is the minimal compose service representation we produce.
type composeService struct {
	Image       string            `yaml:"image,omitempty"`
	Command     interface{}       `yaml:"command,omitempty"`
	Ports       []string          `yaml:"ports,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	Environment []string          `yaml:"environment,omitempty"`
	EnvFile     []string          `yaml:"env_file,omitempty"`
	Networks    []string          `yaml:"networks,omitempty"`
	Restart     string            `yaml:"restart,omitempty"`
	Hostname    string            `yaml:"hostname,omitempty"`
	Labels      []string          `yaml:"labels,omitempty"`
	MemLimit    string            `yaml:"mem_limit,omitempty"`
	Cpus        string            `yaml:"cpus,omitempty"`
	Devices     []string          `yaml:"devices,omitempty"`
	CapAdd      []string          `yaml:"cap_add,omitempty"`
	CapDrop     []string          `yaml:"cap_drop,omitempty"`
	Privileged  bool              `yaml:"privileged,omitempty"`
	User        string            `yaml:"user,omitempty"`
	WorkingDir  string            `yaml:"working_dir,omitempty"`
	Entrypoint  interface{}       `yaml:"entrypoint,omitempty"`
	Extra       map[string]string `yaml:"-"` // unsupported flags collected for comment
}

type composeFile struct {
	Services map[string]*composeService `yaml:"services"`
}

// Composerize converts a `docker run` command string to a compose.yaml string.
// It implements the same flag set as the original composerize npm package.
func Composerize(dockerRunCmd string) (string, error) {
	args, err := splitArgs(dockerRunCmd)
	if err != nil {
		return "", fmt.Errorf("composerize: split args: %w", err)
	}

	// Strip leading "docker run" / "docker container run".
	args = stripDockerRunPrefix(args)

	svc := &composeService{}
	serviceName := "app"
	var positional []string // image + optional command + args

	i := 0
	for i < len(args) {
		arg := args[i]

		// Helper to get the next value, either inline (--flag=val) or as next arg.
		nextVal := func() (string, error) {
			if idx := strings.IndexByte(arg, '='); idx >= 0 {
				return arg[idx+1:], nil
			}
			i++
			if i >= len(args) {
				return "", fmt.Errorf("flag %q requires a value", arg)
			}
			return args[i], nil
		}

		switch {
		// Ignored flags (detach etc.)
		case arg == "-d" || arg == "--detach" ||
			arg == "--rm" || arg == "--pull" || strings.HasPrefix(arg, "--pull=") ||
			arg == "-it" || arg == "-ti" || arg == "-i" || arg == "-t":
			// no-op

		case arg == "--name" || strings.HasPrefix(arg, "--name="):
			v, err := nextVal()
			if err != nil {
				return "", err
			}
			serviceName = v

		case arg == "-p" || arg == "--publish" || strings.HasPrefix(arg, "--publish="):
			v, err := nextVal()
			if err != nil {
				return "", err
			}
			svc.Ports = append(svc.Ports, v)

		case arg == "-v" || arg == "--volume" || strings.HasPrefix(arg, "--volume="):
			v, err := nextVal()
			if err != nil {
				return "", err
			}
			svc.Volumes = append(svc.Volumes, v)

		case arg == "--mount" || strings.HasPrefix(arg, "--mount="):
			// Convert --mount type=bind,source=X,target=Y to X:Y
			v, err := nextVal()
			if err != nil {
				return "", err
			}
			svc.Volumes = append(svc.Volumes, parseMountToVolume(v))

		case arg == "-e" || arg == "--env" || strings.HasPrefix(arg, "--env="):
			v, err := nextVal()
			if err != nil {
				return "", err
			}
			svc.Environment = append(svc.Environment, v)

		case arg == "--env-file" || strings.HasPrefix(arg, "--env-file="):
			v, err := nextVal()
			if err != nil {
				return "", err
			}
			svc.EnvFile = append(svc.EnvFile, v)

		case arg == "--network" || strings.HasPrefix(arg, "--network=") ||
			arg == "--net" || strings.HasPrefix(arg, "--net="):
			v, err := nextVal()
			if err != nil {
				return "", err
			}
			if v != "bridge" && v != "host" && v != "none" {
				svc.Networks = append(svc.Networks, v)
			}

		case arg == "--restart" || strings.HasPrefix(arg, "--restart="):
			v, err := nextVal()
			if err != nil {
				return "", err
			}
			svc.Restart = v

		case arg == "--hostname" || arg == "-h" || strings.HasPrefix(arg, "--hostname="):
			v, err := nextVal()
			if err != nil {
				return "", err
			}
			svc.Hostname = v

		case arg == "--label" || arg == "-l" || strings.HasPrefix(arg, "--label="):
			v, err := nextVal()
			if err != nil {
				return "", err
			}
			svc.Labels = append(svc.Labels, v)

		case arg == "--memory" || arg == "-m" || strings.HasPrefix(arg, "--memory="):
			v, err := nextVal()
			if err != nil {
				return "", err
			}
			svc.MemLimit = v

		case arg == "--cpus" || strings.HasPrefix(arg, "--cpus="):
			v, err := nextVal()
			if err != nil {
				return "", err
			}
			// Validate it's a float.
			if _, err2 := strconv.ParseFloat(v, 64); err2 == nil {
				svc.Cpus = v
			}

		case arg == "--device" || strings.HasPrefix(arg, "--device="):
			v, err := nextVal()
			if err != nil {
				return "", err
			}
			svc.Devices = append(svc.Devices, v)

		case arg == "--cap-add" || strings.HasPrefix(arg, "--cap-add="):
			v, err := nextVal()
			if err != nil {
				return "", err
			}
			svc.CapAdd = append(svc.CapAdd, v)

		case arg == "--cap-drop" || strings.HasPrefix(arg, "--cap-drop="):
			v, err := nextVal()
			if err != nil {
				return "", err
			}
			svc.CapDrop = append(svc.CapDrop, v)

		case arg == "--privileged":
			svc.Privileged = true

		case arg == "--user" || arg == "-u" || strings.HasPrefix(arg, "--user="):
			v, err := nextVal()
			if err != nil {
				return "", err
			}
			svc.User = v

		case arg == "--workdir" || arg == "-w" || strings.HasPrefix(arg, "--workdir="):
			v, err := nextVal()
			if err != nil {
				return "", err
			}
			svc.WorkingDir = v

		case arg == "--entrypoint" || strings.HasPrefix(arg, "--entrypoint="):
			v, err := nextVal()
			if err != nil {
				return "", err
			}
			svc.Entrypoint = v

		case strings.HasPrefix(arg, "--"):
			// Unknown long flag — consume its value if it looks like it takes one.
			if !strings.Contains(arg, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++ // skip value
			}

		case strings.HasPrefix(arg, "-"):
			// Unknown short flag. Docker run short flags are single chars we know
			// about. Anything else (e.g. -s for iperf3 server mode) is an argument
			// to the container entrypoint, not a docker run flag — treat it as a
			// positional argument so it ends up in command:.
			positional = append(positional, arg)

		default:
			// First positional arg is the image; the rest is the command.
			positional = append(positional, arg)
		}
		i++
	}

	// The first positional arg is the image name.
	if len(positional) == 0 {
		return "", fmt.Errorf("composerize: no image specified")
	}
	svc.Image = positional[0]

	// Remaining positionals form the command.
	if len(positional) > 1 {
		if len(positional) == 2 {
			svc.Command = positional[1]
		} else {
			svc.Command = positional[1:]
		}
	}

	cf := composeFile{
		Services: map[string]*composeService{
			serviceName: svc,
		},
	}

	out, err := yaml.Marshal(&cf)
	if err != nil {
		return "", fmt.Errorf("composerize: marshal: %w", err)
	}
	return string(out), nil
}

// splitArgs splits a shell command string into tokens, respecting single and
// double quotes. This mirrors the behaviour of the browser's argument splitting
// that the original composerize library relies on.
func splitArgs(s string) ([]string, error) {
	var args []string
	var cur strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(s); i++ {
		c := rune(s[i])
		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case c == '\\' && !inSingle && i+1 < len(s):
			i++
			cur.WriteByte(s[i])
		case unicode.IsSpace(c) && !inSingle && !inDouble:
			if cur.Len() > 0 {
				args = append(args, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(c)
		}
	}
	if inSingle || inDouble {
		return nil, fmt.Errorf("unterminated quote in command")
	}
	if cur.Len() > 0 {
		args = append(args, cur.String())
	}
	return args, nil
}

func stripDockerRunPrefix(args []string) []string {
	if len(args) == 0 {
		return args
	}
	if args[0] == "docker" {
		args = args[1:]
	}
	if len(args) > 0 && args[0] == "container" {
		args = args[1:]
	}
	if len(args) > 0 && args[0] == "run" {
		args = args[1:]
	}
	return args
}

// parseMountToVolume converts --mount type=bind,source=/x,target=/y → /x:/y
func parseMountToVolume(mount string) string {
	parts := strings.Split(mount, ",")
	src, dst := "", ""
	for _, p := range parts {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch strings.TrimSpace(kv[0]) {
		case "source", "src":
			src = kv[1]
		case "target", "dst", "destination":
			dst = kv[1]
		}
	}
	if src != "" && dst != "" {
		return src + ":" + dst
	}
	return mount
}
