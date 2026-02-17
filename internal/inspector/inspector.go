package inspector

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/alesr/mcpgen/internal/config"
	"github.com/alesr/mcpgen/internal/scaffold"
)

type inspectorCall struct {
	method    string
	extraArgs []string
}

func RunTest(outDir string, cfg *config.Config) error {
	serverName := scaffold.DefaultServerName(cfg.Server.Name)
	methods := make([]inspectorCall, 0)

	if len(cfg.Tools) > 0 {
		methods = append(methods, inspectorCall{method: "tools/list"})
	}

	if len(cfg.Resources) > 0 {
		methods = append(methods, inspectorCall{method: "resources/list"})
	}

	if len(cfg.Prompts) > 0 {
		methods = append(methods, inspectorCall{method: "prompts/list"})
	}

	if len(methods) == 0 {
		return nil
	}

	var methodNames []string
	for _, call := range methods {
		methodNames = append(methodNames, call.method)
	}

	fmt.Printf("Running inspector checks: %s\n", strings.Join(methodNames, ", "))
	fmt.Println("---")

	if cfg.Transport.Type == "http" {
		serverCmd := exec.Command("go", "run", "./cmd/"+serverName)
		serverCmd.Dir = outDir
		serverCmd.Env = append(os.Environ(), "GOWORK=off")
		serverCmd.Stdout = os.Stdout
		serverCmd.Stderr = os.Stderr

		if err := serverCmd.Start(); err != nil {
			return err
		}

		defer func() {
			_ = serverCmd.Process.Kill()
		}()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		if err := waitForPort(ctx, cfg.Transport.HTTPPort); err != nil {
			return fmt.Errorf("could not wait for port: %w", err)
		}
	}

	for i, call := range methods {
		if i > 0 {
			fmt.Println()
		}

		if err := runInspectorCall(outDir, cfg, serverName, call); err != nil {
			return fmt.Errorf("%s failed: %w", call.method, err)
		}
		fmt.Printf("✓ %s\n", call.method)
	}

	fmt.Println("All checks passed.")
	return nil
}

func runInspectorCall(outDir string, cfg *config.Config, serverName string, call inspectorCall) error {
	fmt.Printf("→ %s\n", call.method)

	cmdArgs, err := inspectorArgs(cfg, serverName, call)
	if err != nil {
		return err
	}

	cmd := exec.Command("npx", cmdArgs...)
	cmd.Dir = outDir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func inspectorArgs(cfg *config.Config, serverName string, call inspectorCall) ([]string, error) {
	if cfg.Transport.Type == "http" {
		url := fmt.Sprintf("http://localhost:%d/mcp", cfg.Transport.HTTPPort)
		args := []string{"@modelcontextprotocol/inspector", "--cli", url, "--transport", "http", "--method", call.method}

		if len(call.extraArgs) > 0 {
			args = append(args, call.extraArgs...)
		}
		return args, nil
	}

	args := []string{
		"@modelcontextprotocol/inspector",
		"--cli",
		"--transport",
		"stdio",
		"-e",
		"GOWORK=off",
		"--method",
		call.method,
	}

	if len(call.extraArgs) > 0 {
		args = append(args, call.extraArgs...)
	}

	args = append(args, "--", "go", "run", "./cmd/"+serverName)
	return args, nil
}

func waitForPort(ctx context.Context, port int) error {
	address := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port))
	var d net.Dialer

	for {
		conn, err := d.DialContext(ctx, "tcp", address)
		if err == nil {
			conn.Close()
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("port %d not reachable: %w", port, ctx.Err())
		case <-time.After(500 * time.Millisecond):
			// try again
		}
	}
}
