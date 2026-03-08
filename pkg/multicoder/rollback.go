
package multicoder

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

func shouldSkipPath(path string) bool {
	skipDirs := []string{".git", ".svn", ".hg", ".bzr"}
	
	for _, skipDir := range skipDirs {
		if strings.Contains(path, string(os.PathSeparator)+skipDir+string(os.PathSeparator)) ||
			strings.HasPrefix(path, skipDir+string(os.PathSeparator)) ||
			strings.HasSuffix(path, string(os.PathSeparator)+skipDir) {
			return true
		}
	}
	
	return false
}

func HandleRollback(n *int) error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                     ROLLBACK IN PROGRESS                       ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

	versionFolder, err := GetVersionFolder(n)
	if err != nil {
		return err
	}

	versionNum := "latest"
	if n != nil {
		versionNum = fmt.Sprintf("%d", *n)
	}
	yellow.Printf("→ Rolling back to version: %s\n", versionNum)
	yellow.Printf("  Path: %s\n\n", versionFolder)

	backupFolder := filepath.Join(versionFolder, "backup")

	cyan.Println("→ Restoring files...")
	fileCount := 0
	skippedCount := 0

	err = filepath.Walk(backupFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(backupFolder, path)
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
	green.Printf("✓ Rollback complete: %d file(s) restored to version %s\n", fileCount, versionNum)
	if skippedCount > 0 {
		yellow.Printf("  Skipped %d version control file(s)\n", skippedCount)
	}
	fmt.Println()
	return nil
}
