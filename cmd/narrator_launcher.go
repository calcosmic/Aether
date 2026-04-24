package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
)

const narratorCloseTimeout = 2 * time.Second

var (
	narratorLookPath       = exec.LookPath
	narratorCommandContext = exec.CommandContext
	narratorRuntimePath    = resolveNarratorRuntimePath
)

type narratorLauncher struct {
	cancel     context.CancelFunc
	stdin      io.WriteCloser
	visualPath string
	done       chan error
	stdoutDone chan struct{}

	mu     sync.Mutex
	closed bool
}

func maybeLaunchNarrator(ctx context.Context, root string) *narratorLauncher {
	if !shouldLaunchNarrator() {
		return nil
	}

	nodePath, err := narratorLookPath("node")
	if err != nil {
		return nil
	}
	nodePath, err = filepath.Abs(nodePath)
	if err != nil {
		return nil
	}

	runtimePath, ok := narratorRuntimePath(root)
	if !ok {
		return nil
	}
	runtimePath, err = filepath.Abs(runtimePath)
	if err != nil {
		return nil
	}

	visualPath, err := writeNarratorVisualContract()
	if err != nil {
		return nil
	}

	childCtx, cancel := context.WithCancel(ctx)
	cmd := narratorCommandContext(childCtx, nodePath, runtimePath, "--visuals", visualPath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		_ = os.Remove(visualPath)
		return nil
	}
	childStdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		_ = stdin.Close()
		_ = os.Remove(visualPath)
		return nil
	}
	childStderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		_ = stdin.Close()
		_ = os.Remove(visualPath)
		return nil
	}
	if err := cmd.Start(); err != nil {
		cancel()
		_ = stdin.Close()
		_ = os.Remove(visualPath)
		return nil
	}

	launcher := &narratorLauncher{
		cancel:     cancel,
		stdin:      stdin,
		visualPath: visualPath,
		done:       make(chan error, 1),
		stdoutDone: make(chan struct{}),
	}
	go launcher.copyStdout(childStdout)
	go drainNarratorStderr(childStderr)
	go func() {
		launcher.done <- cmd.Wait()
	}()
	return launcher
}

func shouldLaunchNarrator() bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("AETHER_OUTPUT_MODE")), "json") {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(os.Getenv("AETHER_NARRATOR"))) {
	case "off", "false", "0", "no":
		return false
	case "on", "true", "1", "yes":
		return true
	default:
		return shouldRenderVisualOutput(stdout)
	}
}

func resolveNarratorRuntimePath(root string) (string, bool) {
	candidates := []string{}
	if strings.TrimSpace(root) != "" {
		candidates = append(candidates, filepath.Join(root, ".aether", "ts", "dist", "narrator.js"))
	}
	if hub := strings.TrimSpace(resolveHubPath()); hub != "" {
		candidates = append(candidates, filepath.Join(hub, "system", "ts", "dist", "narrator.js"))
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
	}
	return "", false
}

func writeNarratorVisualContract() (string, error) {
	file, err := os.CreateTemp("", "aether-narrator-visuals-*.json")
	if err != nil {
		return "", err
	}
	defer file.Close()

	payload := map[string]interface{}{
		"castes": casteVisualContracts(),
	}
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(payload); err != nil {
		_ = os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}

func (n *narratorLauncher) EmitEvent(evt events.Event) {
	if n == nil {
		return
	}
	data, err := json.Marshal(evt)
	if err != nil {
		return
	}
	n.writeLine(data)
}

func (n *narratorLauncher) writeLine(data []byte) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.closed || n.stdin == nil {
		return
	}
	if _, err := n.stdin.Write(append(data, '\n')); err != nil {
		n.stdin = nil
	}
}

func (n *narratorLauncher) Close() {
	if n == nil {
		return
	}

	n.mu.Lock()
	if n.closed {
		n.mu.Unlock()
		return
	}
	n.closed = true
	stdin := n.stdin
	n.mu.Unlock()

	if stdin != nil {
		_ = stdin.Close()
	}

	select {
	case <-n.done:
	case <-time.After(narratorCloseTimeout):
		if n.cancel != nil {
			n.cancel()
		}
		select {
		case <-n.done:
		case <-time.After(narratorCloseTimeout):
		}
	}
	if n.cancel != nil {
		n.cancel()
	}
	if n.stdoutDone != nil {
		select {
		case <-n.stdoutDone:
		case <-time.After(narratorCloseTimeout):
		}
	}
	if n.visualPath != "" {
		_ = os.Remove(n.visualPath)
	}
}

func (n *narratorLauncher) copyStdout(r io.Reader) {
	defer close(n.stdoutDone)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r\n")
		if strings.TrimSpace(line) == "" {
			continue
		}
		writeVisualOutput(stdout, line+"\n")
	}
}

func drainNarratorStderr(r io.Reader) {
	_, _ = io.Copy(io.Discard, r)
}
