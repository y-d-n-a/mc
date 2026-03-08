
package shared

import (
	"os"
	"path/filepath"
	"strings"
)

func GetProjectRoot() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}

	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", err
	}

	binDir := filepath.Dir(execPath)
	projectRoot := filepath.Dir(binDir)

	return projectRoot, nil
}

func LoadEnvFile() error {
	projectRoot, err := GetProjectRoot()
	if err != nil {
		return err
	}

	envPath := filepath.Join(projectRoot, ".env")
	data, err := os.ReadFile(envPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}
	}

	return nil
}
