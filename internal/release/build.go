package release

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type BuildResult struct {
	OK     bool   `json:"ok"`
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	Output string `json:"output"`
}

func BuildRuntimeBinary(root string) (BuildResult, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return BuildResult{}, err
	}
	projectRoot := absoluteRoot
	if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err != nil {
		candidate := filepath.Clean(filepath.Join(absoluteRoot, "..", "yeelight-home"))
		if _, candidateErr := os.Stat(filepath.Join(candidate, "go.mod")); candidateErr != nil {
			return BuildResult{}, fmt.Errorf("yeelight-home project not found from %s", absoluteRoot)
		}
		projectRoot = candidate
	}
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	sourcePackage := "./cmd/yeelight-home"
	output := filepath.Join(projectRoot, "bin", fmt.Sprintf("yeelight-home-%s-%s", goos, goarch))
	if goos == "windows" {
		output += ".exe"
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return BuildResult{}, err
	}
	command := exec.Command("go", "build", "-o", output, sourcePackage)
	command.Dir = projectRoot
	data, err := command.CombinedOutput()
	if err != nil {
		return BuildResult{}, fmt.Errorf("go build failed: %w: %s", err, string(data))
	}
	return BuildResult{
		OK:     true,
		OS:     goos,
		Arch:   goarch,
		Output: filepath.ToSlash(output),
	}, nil
}
