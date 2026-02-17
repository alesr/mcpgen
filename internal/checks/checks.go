package checks

import (
	"fmt"
	"os"
	"os/exec"
)

func Run(outDir string) error {
	steps := []struct {
		label string
		cmd   []string
	}{
		{label: "go mod tidy", cmd: []string{"go", "mod", "tidy"}},
		{label: "gofmt -w .", cmd: []string{"gofmt", "-w", "."}},
		{label: "go vet ./...", cmd: []string{"go", "vet", "./..."}},
		{label: "go test ./...", cmd: []string{"go", "test", "./..."}},
	}

	for _, step := range steps {
		fmt.Printf("Running: %s\n", step.label)

		cmd := exec.Command(step.cmd[0], step.cmd[1:]...)

		cmd.Dir = outDir
		cmd.Env = append(os.Environ(), "GOWORK=off") // TODO(alesr): rm gowork
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s failed: %w", step.label, err)
		}
	}
	return nil
}
