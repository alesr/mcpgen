package checks

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/alesr/mcpgen/internal/config"
	"github.com/alesr/mcpgen/internal/pkg/utils"
)

// runInspectorSmoke performs an Inspector v2 CLI smoke call: tools/list plus
// one tools/call when the project has tools. Inspector is a debugging aid;
// the conformance suite is the authoritative gate.
func runInspectorSmoke(ctx context.Context, dir string, cfg *config.Config) (string, string, error) {
	serverName := utils.DefaultServerName(cfg.Server.Name)

	if cfg.Transport.Type != "http" {
		// stdio: inspector launches the server itself via the positional target
		args := []string{
			"--yes", "@modelcontextprotocol/inspector", "--cli",
			"--transport", "stdio",
			"--method", "tools/list",
			"--", "go", "run", "./cmd/" + serverName,
		}
		out, err := runCmd(ctx, dir, "npx", args...)
		if err != nil {
			return out, fmt.Sprintf("inspector v2 CLI smoke failed; run `npx @modelcontextprotocol/inspector --cli --transport stdio --method tools/list -- go run ./cmd/%s` in %s", serverName, dir), err
		}
		return "inspector tools/list ok (stdio)", "", nil
	}

	// http: start the server, then probe it.
	bin := dir + "/dist/mcpgen-check-inspector"
	if out, err := runCmd(ctx, dir, "go", "build", "-o", bin, "./cmd/"+serverName); err != nil {
		return out, fmt.Sprintf("build failed; first error at %s", firstGoLocation(out)), err
	}
	defer func() { _ = os.Remove(bin) }()

	port, err := freePort()
	if err != nil {
		return "", "", err
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	serverCmd := exec.CommandContext(ctx, bin, "-addr", addr)
	serverCmd.Dir = dir
	serverCmd.Env = append(os.Environ(), "GOWORK=off")
	if err := serverCmd.Start(); err != nil {
		return "", "", fmt.Errorf("start server: %w", err)
	}
	defer func() { _ = serverCmd.Process.Kill(); _ = serverCmd.Wait() }()

	url := fmt.Sprintf("http://%s/mcp", addr)
	if err := waitHTTP(ctx, addr); err != nil {
		return "", fmt.Sprintf("server not reachable at %s: %v", url, err), err
	}

	methods := []string{"tools/list"}
	if len(cfg.Tools) > 0 {
		methods = append(methods, "tools/call")
	}
	for _, m := range methods {
		args := []string{"--yes", "@modelcontextprotocol/inspector", "--cli", url, "--transport", "http", "--method", m}
		if m == "tools/call" {
			args = append(args, "--tool-name", cfg.Tools[0].ID, "--tool-args-json", "{}")
		}
		out, err := runCmd(ctx, dir, "npx", args...)
		if err != nil {
			return out, fmt.Sprintf("inspector %s failed; run `npx @modelcontextprotocol/inspector --cli %s --transport http --method %s` in %s", m, url, m, dir), err
		}
	}
	return fmt.Sprintf("inspector smoke ok (%s)", strings.Join(methods, ", ")), "", nil
}

func waitHTTP(ctx context.Context, addr string) error {
	deadline := time.Now().Add(15 * time.Second)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout")
		}
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
}
