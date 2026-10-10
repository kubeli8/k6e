package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/kubeli8/k6e/internal/model"
)

// BuildBinary builds the Go package at packagePath (for example
// "./cmd/control-plane") with the go tool, using repoRoot as the working
// directory, and returns the path of the built executable. The binary is
// written into the test's temporary directory and removed when the test
// finishes.
func BuildBinary(t *testing.T, repoRoot, packagePath string) string {
	t.Helper()

	binaryName := filepath.Base(packagePath)
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	binaryPath := filepath.Join(t.TempDir(), binaryName)

	cmd := exec.Command("go", "build", "-o", binaryPath, packagePath)
	cmd.Dir = repoRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build %s: %v\nOutput: %s", packagePath, err, string(output))
	}

	return binaryPath
}

// Process is a subprocess started by StartProcess with its stdout and stderr
// captured to files, so tests can inspect the output after the process exits.
type Process struct {
	cmd        *exec.Cmd
	stdoutPath string
	stderrPath string
}

// StartProcess starts name with args as a subprocess running in dir, capturing
// stdout and stderr to files inside the test's temporary directory. The
// process is killed and reaped, and the captured logs removed, when the test
// finishes. A failure to create the log files or start the process fails the
// calling test.
func StartProcess(t *testing.T, dir, name string, args ...string) *Process {
	t.Helper()

	logDir := t.TempDir()

	stdout, err := os.CreateTemp(logDir, "stdout-*.log")
	if err != nil {
		t.Fatalf("failed to create stdout log file: %v", err)
	}

	stderr, err := os.CreateTemp(logDir, "stderr-*.log")
	if err != nil {
		stdout.Close()
		t.Fatalf("failed to create stderr log file: %v", err)
	}

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		stdout.Close()
		stderr.Close()
		t.Fatalf("failed to start %s: %v", name, err)
	}

	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		stdout.Close()
		stderr.Close()
	})

	return &Process{
		cmd:        cmd,
		stdoutPath: stdout.Name(),
		stderrPath: stderr.Name(),
	}
}

// DumpLogs writes the subprocess's captured stdout and stderr to the test
// log. It is safe to call from a t.Cleanup while the logs still exist.
func (p *Process) DumpLogs(t *testing.T) {
	t.Helper()

	streams := []struct {
		name string
		path string
	}{
		{name: "stdout", path: p.stdoutPath},
		{name: "stderr", path: p.stderrPath},
	}

	for _, stream := range streams {
		data, err := os.ReadFile(stream.path)
		if err != nil {
			t.Logf("failed to read %s log: %v", stream.name, err)
			continue
		}
		t.Logf("%s log:\n%s", stream.name, data)
	}
}

// WaitUntil polls condition every interval until it returns true, returning
// nil. The condition runs once immediately, before the first interval elapses.
// It returns an error when the condition returns an error or when timeout
// passes without the condition becoming true. interval must be positive.
func WaitUntil(timeout, interval time.Duration, condition func() (bool, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		ok, err := condition()
		if err != nil {
			return fmt.Errorf("polling condition failed: %w", err)
		}
		if ok {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("condition not met within %s", timeout)
		case <-ticker.C:
		}
	}
}

// GetJSON sends a GET request to url and decodes a 2xx JSON response into T.
//
// It returns ok=false with the zero value and no error when the request cannot
// be completed successfully - the server is unreachable, still starting, or
// answered with a non-2xx status - so callers polling during startup can retry
// those conditions. A 2xx response that cannot be decoded as T is a hard error
// returned in err, wrapped with the request URL.
func GetJSON[T any](client *http.Client, url string) (value T, ok bool, err error) {
	resp, err := client.Get(url)
	if err != nil {
		return value, false, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return value, false, nil
	}

	if err := json.NewDecoder(resp.Body).Decode(&value); err != nil {
		return value, false, fmt.Errorf("GET %s: failed to decode response: %w", url, err)
	}

	return value, true, nil
}

// DoJSON sends an HTTP request with an optional JSON body to url and decodes
// the JSON response body into T. Any failure - marshalling, sending, a non-2xx
// status (reported together with the response body), or decoding - fails the
// calling test with a message naming the request.
func DoJSON[T any](t *testing.T, client *http.Client, method, url string, body any) T {
	t.Helper()

	var reqBody io.Reader

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("%s %s: failed to marshal request body: %v", method, url, err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("%s %s: failed to create request: %v", method, url, err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: request failed: %v", method, url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		t.Fatalf(
			"%s %s: unexpected status %d: %s",
			method,
			url,
			resp.StatusCode,
			bytes.TrimSpace(responseBody),
		)
	}

	var result T

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("%s %s: failed to decode response: %v", method, url, err)
	}

	return result
}

const (
	nodeReadyTimeout = 30 * time.Second
	pollInterval     = 500 * time.Millisecond
)

// WaitForReadyNode polls the control plane's node list until a node with ID
// nodeID reports status Ready, failing the calling test if it does not within
// 30 seconds. Unreachable servers and non-2xx responses are tolerated while
// the control plane is still starting up.
func WaitForReadyNode(t *testing.T, client *http.Client, baseURL, nodeID string) {
	t.Helper()

	url := baseURL + "/api/v1/nodes"

	err := WaitUntil(nodeReadyTimeout, pollInterval, func() (bool, error) {
		nodes, ok, err := GetJSON[[]model.Node](client, url)
		if err != nil || !ok {
			return false, err
		}

		for _, node := range nodes {
			if node.ID == nodeID && node.Status == model.NodeStatusReady {
				return true, nil
			}
		}

		return false, nil
	})
	if err != nil {
		t.Fatalf("timed out waiting for node %q to become Ready: %v", nodeID, err)
	}
}
