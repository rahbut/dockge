package stack

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/rahbut/dockge/backend/models"
	"github.com/rahbut/dockge/backend/terminal"
)

// Status codes — mirrors util-common.ts.
const (
	StatusUnknown      = 0
	StatusCreatedFile  = 1 // directory + compose file exists, never run
	StatusCreatedStack = 2 // created in Docker
	StatusRunning      = 3
	StatusExited       = 4
)

// AcceptedComposeFileNames lists compose file names in priority order.
var AcceptedComposeFileNames = []string{
	"compose.yaml",
	"docker-compose.yaml",
	"docker-compose.yml",
	"compose.yml",
}

var stackNameRE = regexp.MustCompile(`^[a-z0-9_-]+$`)

// ─── Stack ────────────────────────────────────────────────────────────────────

// Stack represents a single Docker Compose project managed by Dockge.
type Stack struct {
	Name            string
	Status          int
	ComposeYAML     string
	ComposeENV      string
	ComposeFileName string
	ConfigFilePath  string
	UpdateAvailable *bool
	UpdateDetails   map[string]ServiceUpdateResult

	stacksDir string // parent directory for all stacks
	endpoint  string // local endpoint (empty string) or agent endpoint
}

// ServiceUpdateResult is the per-service image update check result.
type ServiceUpdateResult struct {
	Image           string `json:"image"`
	UpdateAvailable bool   `json:"updateAvailable"`
	Error           string `json:"error,omitempty"`
}

// ─── Managed stack cache ──────────────────────────────────────────────────────

var (
	managedCacheMu sync.Mutex
	managedCache   map[string]*Stack
)

// ─── Constructor ─────────────────────────────────────────────────────────────

// New creates a Stack, resolving the compose file name from disk unless
// skipFS is true.
func New(stacksDir, name, composeYAML, composeENV string, skipFS bool) *Stack {
	s := &Stack{
		Name:            name,
		ComposeYAML:     composeYAML,
		ComposeENV:      composeENV,
		ComposeFileName: "compose.yaml",
		stacksDir:       stacksDir,
	}
	if !skipFS {
		for _, fn := range AcceptedComposeFileNames {
			if _, err := os.Stat(filepath.Join(s.Path(), fn)); err == nil {
				s.ComposeFileName = fn
				break
			}
		}
	}
	return s
}

// Path returns the absolute directory path for this stack.
func (s *Stack) Path() string {
	return filepath.Join(s.stacksDir, s.Name)
}

// IsManagedByDockge reports whether the stack directory exists on disk.
func (s *Stack) IsManagedByDockge() bool {
	info, err := os.Stat(s.Path())
	return err == nil && info.IsDir()
}

// ─── Validation ──────────────────────────────────────────────────────────────

// Validate checks the stack name, YAML, and .env format.
func (s *Stack) Validate() error {
	if !stackNameRE.MatchString(s.Name) {
		return fmt.Errorf("stack name can only contain [a-z][0-9] _ -")
	}
	var doc interface{}
	if err := yaml.Unmarshal([]byte(s.ComposeYAML), &doc); err != nil {
		return fmt.Errorf("invalid compose YAML: %w", err)
	}
	lines := strings.Split(s.ComposeENV, "\n")
	if len(lines) == 1 && !strings.Contains(lines[0], "=") && len(lines[0]) > 0 {
		return fmt.Errorf("invalid .env format")
	}
	return nil
}

// ─── File I/O ─────────────────────────────────────────────────────────────────

// Load reads ComposeYAML and ComposeENV from disk.
func (s *Stack) Load() {
	if b, err := os.ReadFile(filepath.Join(s.Path(), s.ComposeFileName)); err == nil {
		s.ComposeYAML = string(b)
	}
	if b, err := os.ReadFile(filepath.Join(s.Path(), ".env")); err == nil {
		s.ComposeENV = string(b)
	}
}

// Save writes compose.yaml and .env to disk.
func (s *Stack) Save(isAdd bool) error {
	if err := s.Validate(); err != nil {
		return err
	}
	dir := s.Path()
	if isAdd {
		if _, err := os.Stat(dir); err == nil {
			return fmt.Errorf("stack name already exists")
		}
		if err := os.Mkdir(dir, 0o755); err != nil {
			return fmt.Errorf("create stack dir: %w", err)
		}
	} else {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return fmt.Errorf("stack not found")
		}
	}

	if err := os.WriteFile(filepath.Join(dir, s.ComposeFileName), []byte(s.ComposeYAML), 0o644); err != nil {
		return fmt.Errorf("write compose file: %w", err)
	}

	envPath := filepath.Join(dir, ".env")
	_, envExists := os.Stat(envPath)
	if envExists == nil || strings.TrimSpace(s.ComposeENV) != "" {
		if err := os.WriteFile(envPath, []byte(s.ComposeENV), 0o644); err != nil {
			return fmt.Errorf("write .env: %w", err)
		}
	}
	return nil
}

// ─── Docker Compose operations ────────────────────────────────────────────────

// getComposeOptions builds the `docker compose <cmd>` argument slice, prepending
// any --env-file options that exist on disk.
func (s *Stack) getComposeOptions(cmd string, extra ...string) []string {
	opts := []string{"compose", cmd}
	opts = append(opts, extra...)

	globalEnv := filepath.Join(s.stacksDir, "global.env")
	stackEnv := filepath.Join(s.Path(), ".env")

	if _, err := os.Stat(globalEnv); err == nil {
		// Prepend at index 1 (after "compose").
		if _, err2 := os.Stat(stackEnv); err2 == nil {
			opts = insertAt(opts, 1, "--env-file", "./.env")
		}
		opts = insertAt(opts, 1, "--env-file", "../global.env")
	}
	return opts
}

func insertAt(slice []string, idx int, vals ...string) []string {
	out := make([]string, 0, len(slice)+len(vals))
	out = append(out, slice[:idx]...)
	out = append(out, vals...)
	out = append(out, slice[idx:]...)
	return out
}

// terminalName returns the compose operation terminal name.
// Format mirrors util-common.ts getComposeTerminalName:
//
//	"compose-" + endpoint + "-" + stackName
//
// When endpoint is "" this produces "compose--stackName" (note double dash).
func (s *Stack) terminalName() string {
	return fmt.Sprintf("compose-%s-%s", s.endpoint, s.Name)
}

// combinedTerminalName mirrors util-common.ts getCombinedTerminalName:
//
//	"combined-" + endpoint + "-" + stackName
func combinedTerminalName(endpoint, stackName string) string {
	return fmt.Sprintf("combined-%s-%s", endpoint, stackName)
}

// containerExecTerminalName mirrors util-common.ts getContainerExecTerminalName:
//
//	"container-exec-" + endpoint + "-" + stackName + "-" + container + "-" + index
func containerExecTerminalName(endpoint, stackName, serviceName string, index int) string {
	return fmt.Sprintf("container-exec-%s-%s-%s-%d", endpoint, stackName, serviceName, index)
}

// Deploy runs `docker compose up -d --remove-orphans`.
func (s *Stack) Deploy(socket terminal.Emitter) error {
	code, err := terminal.Exec(socket, s.terminalName(), "docker",
		s.getComposeOptions("up", "-d", "--remove-orphans"), s.Path())
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("deploy failed (exit %d)", code)
	}
	return nil
}

// Start runs `docker compose up -d --remove-orphans`.
func (s *Stack) Start(socket terminal.Emitter) error {
	code, err := terminal.Exec(socket, s.terminalName(), "docker",
		s.getComposeOptions("up", "-d", "--remove-orphans"), s.Path())
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("start failed (exit %d)", code)
	}
	return nil
}

// Stop runs `docker compose stop`.
func (s *Stack) Stop(socket terminal.Emitter) error {
	code, err := terminal.Exec(socket, s.terminalName(), "docker",
		s.getComposeOptions("stop"), s.Path())
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("stop failed (exit %d)", code)
	}
	return nil
}

// Restart runs `docker compose restart`.
func (s *Stack) Restart(socket terminal.Emitter) error {
	code, err := terminal.Exec(socket, s.terminalName(), "docker",
		s.getComposeOptions("restart"), s.Path())
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("restart failed (exit %d)", code)
	}
	return nil
}

// Down runs `docker compose down`.
// For orphaned stacks (no compose file on disk), it uses `-p projectname`
// instead of `-f composefile` so docker can clean up by project name alone.
func (s *Stack) Down(socket terminal.Emitter) error {
	var args []string
	if !s.IsManagedByDockge() {
		// No directory — use project name flag so docker can find the project
		// without needing the original compose file.
		args = []string{"compose", "-p", s.Name, "down"}
	} else {
		args = s.getComposeOptions("down")
	}
	code, err := terminal.Exec(socket, s.terminalName(), "docker", args, s.Path())
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("down failed (exit %d)", code)
	}
	return nil
}

// Delete runs `docker compose down --remove-orphans` then removes the directory.
func (s *Stack) Delete(socket terminal.Emitter) error {
	code, err := terminal.Exec(socket, s.terminalName(), "docker",
		s.getComposeOptions("down", "--remove-orphans"), s.Path())
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("delete failed (exit %d)", code)
	}
	return os.RemoveAll(s.Path())
}

// Update pulls images (and rebuilds if needed), then re-deploys if running.
func (s *Stack) Update(socket terminal.Emitter) error {
	code, err := terminal.Exec(socket, s.terminalName(), "docker",
		s.getComposeOptions("pull"), s.Path())
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("pull failed (exit %d)", code)
	}

	if s.HasBuildServices() {
		code, err = terminal.Exec(socket, s.terminalName(), "docker",
			s.getComposeOptions("build", "--pull"), s.Path())
		if err != nil {
			return err
		}
		if code != 0 {
			return fmt.Errorf("build failed (exit %d)", code)
		}
	}

	if err := s.UpdateStatus(); err != nil || s.Status != StatusRunning {
		return err
	}

	code, err = terminal.Exec(socket, s.terminalName(), "docker",
		s.getComposeOptions("up", "-d", "--remove-orphans"), s.Path())
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("restart after update failed (exit %d)", code)
	}

	s.UpdateAvailable = nil
	s.UpdateDetails = nil
	return nil
}

// HasBuildServices reports whether any service in the compose YAML uses build:.
func (s *Stack) HasBuildServices() bool {
	var doc struct {
		Services map[string]map[string]interface{} `yaml:"services"`
	}
	if err := yaml.Unmarshal([]byte(s.ComposeYAML), &doc); err != nil {
		return false
	}
	for _, svc := range doc.Services {
		if _, ok := svc["build"]; ok {
			return true
		}
	}
	return false
}

// ─── Status ───────────────────────────────────────────────────────────────────

// UpdateStatus refreshes the in-memory Status field from `docker compose ls`.
func (s *Stack) UpdateStatus() error {
	list, err := GetStatusList()
	if err != nil {
		return err
	}
	if st, ok := list[s.Name]; ok {
		s.Status = st
	} else {
		s.Status = StatusUnknown
	}
	return nil
}

// GetStatusList runs `docker compose ls --all --format json` and returns a
// map of stack name → status code.
func GetStatusList() (map[string]int, error) {
	out, err := exec.Command("docker", "compose", "ls", "--all", "--format", "json").Output()
	if err != nil {
		return nil, err
	}
	var list []struct {
		Name   string `json:"Name"`
		Status string `json:"Status"`
	}
	if err := json.Unmarshal(out, &list); err != nil {
		return nil, err
	}
	result := make(map[string]int, len(list))
	for _, item := range list {
		result[item.Name] = StatusConvert(item.Status)
	}
	return result, nil
}

// StatusConvert converts the `docker compose ls` status string to an int code.
func StatusConvert(status string) int {
	switch {
	case strings.HasPrefix(status, "created"):
		return StatusCreatedStack
	case strings.Contains(status, "exited"):
		return StatusExited
	case strings.HasPrefix(status, "running"):
		return StatusRunning
	default:
		return StatusUnknown
	}
}

// ─── Stack list ───────────────────────────────────────────────────────────────

// GetStackList scans stacksDir and combines with `docker compose ls` output.
func GetStackList(stacksDir string, useCache bool) (map[string]*Stack, error) {
	managedCacheMu.Lock()
	if useCache && len(managedCache) > 0 {
		out := make(map[string]*Stack, len(managedCache))
		for k, v := range managedCache {
			out[k] = v
		}
		managedCacheMu.Unlock()
		return out, nil
	}
	managedCacheMu.Unlock()

	stackList := make(map[string]*Stack)

	entries, err := os.ReadDir(stacksDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		hasCompose := false
		for _, fn := range AcceptedComposeFileNames {
			if _, err2 := os.Stat(filepath.Join(stacksDir, name, fn)); err2 == nil {
				hasCompose = true
				break
			}
		}
		if !hasCompose {
			continue
		}
		s := New(stacksDir, name, "", "", false)
		s.Status = StatusCreatedFile
		stackList[name] = s
	}

	// Merge `docker compose ls` status — includes stacks with no directory.
	out, err := exec.Command("docker", "compose", "ls", "--all", "--format", "json").Output()
	if err == nil {
		var composeList []struct {
			Name        string `json:"Name"`
			Status      string `json:"Status"`
			ConfigFiles string `json:"ConfigFiles"`
		}
		if json.Unmarshal(out, &composeList) == nil {
			for _, item := range composeList {
				s, exists := stackList[item.Name]
				if !exists {
					if item.Name == "dockge" {
						continue
					}
					s = New(stacksDir, item.Name, "", "", true)
					stackList[item.Name] = s
				}
				s.Status = StatusConvert(item.Status)
				s.ConfigFilePath = item.ConfigFiles
			}
		}
	}

	// Cache after the docker compose merge so docker-only stacks are included.
	managedCacheMu.Lock()
	managedCache = make(map[string]*Stack, len(stackList))
	for k, v := range stackList {
		managedCache[k] = v
	}
	managedCacheMu.Unlock()

	return stackList, nil
}

// GetStack returns a specific stack by name.
func GetStack(stacksDir, name string, skipFS bool) (*Stack, error) {
	dir := filepath.Join(stacksDir, name)
	if !skipFS {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			// Not in stacksDir — check if it's a docker-managed stack.
			// Use useCache=false so docker-only stacks (no local directory)
			// are included in the lookup.
			list, err2 := GetStackList(stacksDir, false)
			if err2 != nil {
				return nil, err2
			}
			if s, ok := list[name]; ok {
				return s, nil
			}
			return nil, fmt.Errorf("stack not found")
		}
	}
	s := New(stacksDir, name, "", "", skipFS)
	s.Status = StatusUnknown
	if !skipFS {
		s.ConfigFilePath = filepath.Clean(dir)
	}
	return s, nil
}

// ─── Service status ──────────────────────────────────────────────────────────

// ServiceStatus holds per-container state and port mappings.
type ServiceStatus struct {
	State string   `json:"state"`
	Ports []string `json:"ports"`
}

// GetServiceStatusList runs `docker compose ps --format json` and parses
// the NDJSON output (one JSON object per line).
func (s *Stack) GetServiceStatusList() (map[string]ServiceStatus, error) {
	cmd := exec.Command("docker", s.getComposeOptions("ps", "--format", "json")...)
	cmd.Dir = s.Path()
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	result := make(map[string]ServiceStatus)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var obj struct {
			Service string `json:"Service"`
			State   string `json:"State"`
			Health  string `json:"Health"`
			Ports   string `json:"Ports"`
		}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}
		var ports []string
		for _, p := range strings.Split(obj.Ports, ", ") {
			p = strings.TrimSpace(p)
			if strings.Contains(p, "->") &&
				!strings.HasPrefix(p, ":::") &&
				!strings.HasPrefix(p, "[::]") {
				ports = append(ports, p)
			}
		}
		state := obj.State
		if obj.Health != "" {
			state = obj.Health
		}
		result[obj.Service] = ServiceStatus{State: state, Ports: ports}
	}
	return result, scanner.Err()
}

// ─── Image update checks ──────────────────────────────────────────────────────

// CheckUpdates checks each service image against its registry and stores the
// result on the stack.
func (s *Stack) CheckUpdates() (map[string]ServiceUpdateResult, error) {
	var doc struct {
		Services map[string]map[string]interface{} `yaml:"services"`
	}
	if err := yaml.Unmarshal([]byte(s.ComposeYAML), &doc); err != nil {
		return nil, nil
	}

	results := make(map[string]ServiceUpdateResult)

	for svcName, svc := range doc.Services {
		image, _ := svc["image"].(string)
		if image == "" {
			results[svcName] = ServiceUpdateResult{
				Image: "(local build)", Error: "localBuild",
			}
			continue
		}

		// Get local digest.
		localOut, err := exec.Command("docker", "image", "inspect",
			"--format", "{{index .RepoDigests 0}}", image).Output()
		if err != nil {
			results[svcName] = ServiceUpdateResult{Image: image, Error: "notPulled"}
			continue
		}
		localDigest := strings.TrimSpace(string(localOut))
		localSHA := localDigest
		if idx := strings.Index(localDigest, "@"); idx >= 0 {
			localSHA = localDigest[idx+1:]
		}
		if localSHA == "" {
			results[svcName] = ServiceUpdateResult{Image: image, Error: "noLocalDigest"}
			continue
		}

		reg, repo, tag := ParseImageRef(image)
		remoteDigest, err := GetRegistryDigest(reg, repo, tag)
		if err != nil || remoteDigest == "" {
			results[svcName] = ServiceUpdateResult{Image: image, Error: "registryError"}
			continue
		}

		results[svcName] = ServiceUpdateResult{
			Image:           image,
			UpdateAvailable: localSHA != remoteDigest,
		}
	}

	s.UpdateDetails = results
	hasUpdate := false
	for _, r := range results {
		if r.UpdateAvailable {
			hasUpdate = true
			break
		}
	}
	s.UpdateAvailable = &hasUpdate

	return results, nil
}

// ─── Terminal joining ─────────────────────────────────────────────────────────

// JoinCombinedTerminal subscribes socket to the combined `docker compose logs`
// terminal for this stack.
//
// If an existing terminal is still running (containers are up), reuse it so
// all connected clients share the same stream. If it has exited (e.g. all
// containers stopped), create a fresh one so the historical logs are replayed.
func (s *Stack) JoinCombinedTerminal(socket terminal.Emitter) {
	name := combinedTerminalName(s.endpoint, s.Name)

	t := terminal.GetTerminal(name)
	if t == nil {
		// No running terminal — create a fresh one.
		t = terminal.NewTerminal(name, "docker",
			s.getComposeOptions("logs", "-f", "--tail", "100"), s.Path())
		t.EnableKeepAlive = true
		t.SetRows(terminal.CombinedTerminalRows)
		t.SetCols(terminal.CombinedTerminalCols)
		t.Join(socket)
		t.Start()
	} else {
		// Reuse the running terminal — send the buffered history then subscribe.
		t.Join(socket)
		// Replay buffered output to the newly joined socket immediately.
		if buf := t.GetBuffer(); buf != "" {
			socket.EmitAgent("terminalWrite", name, buf)
		}
	}
}

// LeaveCombinedTerminal unsubscribes socket from the combined terminal.
func (s *Stack) LeaveCombinedTerminal(socket terminal.Emitter) {
	name := combinedTerminalName(s.endpoint, s.Name)
	if t := terminal.GetTerminal(name); t != nil {
		t.Leave(socket)
	}
}

// JoinContainerTerminal subscribes socket to a `docker compose exec` terminal.
func (s *Stack) JoinContainerTerminal(socket terminal.Emitter, serviceName, shell string, index int) {
	name := containerExecTerminalName(s.endpoint, s.Name, serviceName, index)
	t := terminal.GetTerminal(name)
	if t == nil {
		it := terminal.NewInteractiveTerminal(name, "docker",
			s.getComposeOptions("exec", serviceName, shell), s.Path())
		it.SetRows(terminal.TerminalRows)
		t = it.Terminal
	}
	t.Join(socket)
	t.Start()
}

// ─── JSON serialisation ───────────────────────────────────────────────────────

// ToSimpleJSON returns a lean representation for the stack list broadcast.
func (s *Stack) ToSimpleJSON() map[string]interface{} {
	// Emit an untyped nil for updateAvailable when it hasn't been checked yet.
	// If we place the *bool pointer directly into a map[string]interface{}, Go
	// stores a typed nil ((*bool)(nil)) which is NOT equal to the untyped nil
	// checked in BroadcastStackList — causing the DB-cached value to be skipped
	// and every 10-second broadcast to reset badges to null on the client.
	var ua interface{}
	if s.UpdateAvailable != nil {
		ua = *s.UpdateAvailable
	}
	return map[string]interface{}{
		"name":              s.Name,
		"status":            s.Status,
		"tags":              []interface{}{},
		"isManagedByDockge": s.IsManagedByDockge(),
		"composeFileName":   s.ComposeFileName,
		"endpoint":          s.endpoint,
		"updateAvailable":   ua,
	}
}

// ToJSON returns the full stack representation including YAML content and
// primary hostname.
func (s *Stack) ToJSON(ctx context.Context, endpoint string) (map[string]interface{}, error) {
	primaryHostname, _ := models.GetPrimaryHostname(ctx)
	if primaryHostname == "" {
		if endpoint == "" {
			primaryHostname = "localhost"
		} else {
			// Best-effort: use host portion of endpoint.
			primaryHostname = endpoint
		}
	}

	obj := s.ToSimpleJSON()
	obj["composeYAML"] = s.ComposeYAML
	obj["composeENV"] = s.ComposeENV
	obj["primaryHostname"] = primaryHostname
	obj["updateDetails"] = s.UpdateDetails
	return obj, nil
}
