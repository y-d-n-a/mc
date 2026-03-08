
package multicoder

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

const CheckpointFile = ".checkpoint"

func SetCheckpoint() error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                    SETTING CHECKPOINT                          ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

	yellow.Println("→ Getting latest version...")
	versionFolder, err := GetLatestVersionFolder()
	if err != nil {
		return fmt.Errorf("failed to get latest version: %v", err)
	}

	versionNum := filepath.Base(versionFolder)
	versionNumStr := versionNum[7:]

	yellow.Println("→ Creating checkpoint snapshot of current state...")
	checkpointFolder := filepath.Join(versionFolder, "checkpoint")
	if err := os.RemoveAll(checkpointFolder); err != nil {
		return fmt.Errorf("failed to clear old checkpoint folder: %v", err)
	}
	if err := os.MkdirAll(checkpointFolder, 0755); err != nil {
		return fmt.Errorf("failed to create checkpoint folder: %v", err)
	}

	fileCount := 0
	skippedCount := 0
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %v", err)
	}

	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.Contains(path, WorkspaceDir) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if shouldSkipPath(path) {
			if info.IsDir() {
				skippedCount++
				return filepath.SkipDir
			}
			skippedCount++
			return nil
		}

		if info.IsDir() {
			return nil
		}

		absFile, err := filepath.Abs(path)
		if err != nil {
			return err
		}

		var checkpointFilePath string
		if strings.HasPrefix(absFile, cwd+string(os.PathSeparator)) || absFile == cwd {
			relPath, _ := filepath.Rel(cwd, absFile)
			checkpointFilePath = filepath.Join(checkpointFolder, relPath)
		} else {
			relPath, _ := filepath.Rel("/", absFile)
			checkpointFilePath = filepath.Join(checkpointFolder, "external", relPath)
		}

		if err := os.MkdirAll(filepath.Dir(checkpointFilePath), 0755); err != nil {
			return err
		}

		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.Create(checkpointFilePath)
		if err != nil {
			return err
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			return err
		}

		fileCount++
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create checkpoint snapshot: %v", err)
	}

	green.Printf("  ✓ Saved %d file(s) to checkpoint\n", fileCount)
	if skippedCount > 0 {
		yellow.Printf("  ✓ Skipped %d version control file(s)\n", skippedCount)
	}
	fmt.Println()

	checkpointPath := filepath.Join(WorkspaceDir, CheckpointFile)
	if err := os.WriteFile(checkpointPath, []byte(versionNumStr), 0644); err != nil {
		return fmt.Errorf("failed to write checkpoint file: %v", err)
	}

	green.Printf("✓ Checkpoint set to version %s\n", versionNumStr)
	green.Printf("  Path: %s\n\n", versionFolder)

	return nil
}

func GetCheckpoint() (int, error) {
	checkpointPath := filepath.Join(WorkspaceDir, CheckpointFile)
	
	data, err := os.ReadFile(checkpointPath)
	if err != nil {
		if os.IsNotExist(err) {
			return -1, fmt.Errorf("no checkpoint set. Use 'mc checkpoint' to set one")
		}
		return -1, fmt.Errorf("failed to read checkpoint file: %v", err)
	}

	versionNum, err := strconv.Atoi(string(data))
	if err != nil {
		return -1, fmt.Errorf("invalid checkpoint data: %v", err)
	}

	return versionNum, nil
}

func RollbackToCheckpoint() error {
	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                ROLLBACK TO CHECKPOINT                          ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

	yellow.Println("→ Reading checkpoint...")
	checkpointVersion, err := GetCheckpoint()
	if err != nil {
		return err
	}

	return HandleRollbackToCheckpoint(checkpointVersion)
}

func HandleRollbackToCheckpoint(versionNum int) error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	versionFolder := filepath.Join(WorkspaceDir, "versions", fmt.Sprintf("version%d", versionNum))
	checkpointFolder := filepath.Join(versionFolder, "checkpoint")

	if _, err := os.Stat(checkpointFolder); os.IsNotExist(err) {
		return fmt.Errorf("no checkpoint snapshot found in version %d", versionNum)
	}

	yellow.Printf("→ Rolling back to checkpoint in version: %d\n", versionNum)
	yellow.Printf("  Path: %s\n\n", checkpointFolder)

	cyan.Println("→ Restoring files...")
	fileCount := 0
	skippedCount := 0

	err := filepath.Walk(checkpointFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(checkpointFolder, path)
		if err != nil {
			return err
		}

		if shouldSkipPath(relativePath) {
			skippedCount++
			return nil
		}

		var originalFilePath string
		if strings.HasPrefix(relativePath, "external"+string(os.PathSeparator)) {
			originalFilePath = filepath.Join("/", relativePath[9:])
		} else {
			cwd, _ := os.Getwd()
			originalFilePath = filepath.Join(cwd, relativePath)
		}

		if shouldSkipPath(originalFilePath) {
			skippedCount++
			return nil
		}

		if err := os.MkdirAll(filepath.Dir(originalFilePath), 0755); err != nil {
			return err
		}

		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.Create(originalFilePath)
		if err != nil {
			return err
		}
		defer dst.Close()

		_, err = io.Copy(dst, src)
		if err == nil {
			fileCount++
			green.Printf("  ✓ Restored: %s\n", originalFilePath)
		}
		return err
	})

	if err != nil {
		return err
	}

	fmt.Println()
	green.Printf("✓ Rollback complete: %d file(s) restored to checkpoint in version %d\n", fileCount, versionNum)
	if skippedCount > 0 {
		yellow.Printf("  Skipped %d version control file(s)\n", skippedCount)
	}
	fmt.Println()
	return nil
}
