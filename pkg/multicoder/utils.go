
package multicoder

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

const WorkspaceDir = ".mcoder-workspace"

func CreateVersionFolder() (string, error) {
	versionsDir := filepath.Join(WorkspaceDir, "versions")
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		return "", err
	}

	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return "", err
	}

	maxVersion := -1
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "version") {
			numStr := strings.TrimPrefix(entry.Name(), "version")
			if num, err := strconv.Atoi(numStr); err == nil && num > maxVersion {
				maxVersion = num
			}
		}
	}

	newVersion := maxVersion + 1
	newVersionFolder := filepath.Join(versionsDir, fmt.Sprintf("version%d", newVersion))
	if err := os.MkdirAll(newVersionFolder, 0755); err != nil {
		return "", err
	}

	return newVersionFolder, nil
}

func SaveResponse(responseFolder string, index int, response string) error {
	responseFile := filepath.Join(responseFolder, fmt.Sprintf("response%d.txt", index))
	return os.WriteFile(responseFile, []byte(response), 0644)
}

func GetLatestVersionFolder() (string, error) {
	versionsDir := filepath.Join(WorkspaceDir, "versions")
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return "", err
	}

	maxVersion := -1
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "version") {
			numStr := strings.TrimPrefix(entry.Name(), "version")
			if num, err := strconv.Atoi(numStr); err == nil && num > maxVersion {
				maxVersion = num
			}
		}
	}

	if maxVersion == -1 {
		return "", fmt.Errorf("no version folders found")
	}

	return filepath.Join(versionsDir, fmt.Sprintf("version%d", maxVersion)), nil
}

func ReadResponseFile(responseFile string) (string, error) {
	data, err := os.ReadFile(responseFile)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func GetVersionFolder(n *int) (string, error) {
	if n == nil {
		return GetLatestVersionFolder()
	}
	return filepath.Join(WorkspaceDir, "versions", fmt.Sprintf("version%d", *n)), nil
}

func ClearWorkspace(confirm bool) error {
	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow, color.Bold)
	green := color.New(color.FgGreen)

	if !confirm {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println()
		yellow.Println("⚠ WARNING: THIS WILL DELETE ALL MCODER WORKSPACE FILES")
		fmt.Print("Proceed? (Y/n): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if input != "y" {
			cyan.Println("Operation cancelled.")
			return nil
		}
	}

	if err := os.RemoveAll(WorkspaceDir); err != nil {
		return err
	}

	green.Printf("\n✓ %s has been cleared.\n\n", WorkspaceDir)
	return nil
}

func BackupCurrentState() error {
	backupFolder := filepath.Join(WorkspaceDir, "last-write-backup")
	if err := os.RemoveAll(backupFolder); err != nil {
		return err
	}
	if err := os.MkdirAll(backupFolder, 0755); err != nil {
		return err
	}

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.Contains(path, WorkspaceDir) {
			return nil
		}

		if shouldSkipPath(path) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		backupPath := filepath.Join(backupFolder, path)
		if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
			return err
		}

		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.Create(backupPath)
		if err != nil {
			return err
		}
		defer dst.Close()

		_, err = io.Copy(dst, src)
		return err
	})

	return err
}

func UndoLastWrite() error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                       UNDO LAST WRITE                          ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

	backupFolder := filepath.Join(WorkspaceDir, "last-write-backup")
	if _, err := os.Stat(backupFolder); os.IsNotExist(err) {
		return fmt.Errorf("no backup found to undo")
	}

	yellow.Println("→ Restoring files from backup...")
	fileCount := 0
	skippedCount := 0

	err := filepath.Walk(backupFolder, func(path string, info os.FileInfo, err error) error {
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

		originalPath := filepath.Join(".", relativePath)
		if err := os.MkdirAll(filepath.Dir(originalPath), 0755); err != nil {
			return err
		}

		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.Create(originalPath)
		if err != nil {
			return err
		}
		defer dst.Close()

		_, err = io.Copy(dst, src)
		if err == nil {
			fileCount++
			green.Printf("  ✓ Restored: %s\n", originalPath)
		}
		return err
	})

	if err != nil {
		return err
	}

	fmt.Println()
	green.Printf("✓ Undo complete: %d file(s) restored\n", fileCount)
	if skippedCount > 0 {
		yellow.Printf("  Skipped %d version control file(s)\n", skippedCount)
	}
	fmt.Println()
	return nil
}

func GetUserInstructionsFromEditor() (string, error) {
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)

	if err := os.MkdirAll(WorkspaceDir, 0755); err != nil {
		return "", err
	}

	tempFile := filepath.Join(WorkspaceDir, "temp_instructions.txt")
	defer os.Remove(tempFile)

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nvim"
	}

	initialContent := ""
	if err := os.WriteFile(tempFile, []byte(initialContent), 0644); err != nil {
		return "", err
	}

	cmd := exec.Command(editor, tempFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor failed: %v", err)
	}

	data, err := os.ReadFile(tempFile)
	if err != nil {
		return "", err
	}

	content := strings.TrimSpace(string(data))

	if content == "" || content == initialContent {
		red.Println("\n✗ No instructions provided. Operation cancelled.\n")
		return "", fmt.Errorf("no instructions provided")
	}

	yellow.Printf("  ✓ Instructions captured (%d bytes)\n\n", len(content))

	return content, nil
}

func ReadMcignore() ([]string, error) {
	mcignorePath := filepath.Join(WorkspaceDir, ".mcignore")

	if _, err := os.Stat(mcignorePath); os.IsNotExist(err) {
		if err := os.MkdirAll(WorkspaceDir, 0755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(mcignorePath, []byte("__pycache__\n"), 0644); err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(mcignorePath)
	if err != nil {
		return nil, err
	}

	var patterns []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}

	return patterns, nil
}

func Ignore(pattern string) error {
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	mcignorePath := filepath.Join(WorkspaceDir, ".mcignore")
	if err := os.MkdirAll(WorkspaceDir, 0755); err != nil {
		return err
	}

	var lines []string
	if data, err := os.ReadFile(mcignorePath); err == nil {
		lines = strings.Split(string(data), "\n")
	} else {
		lines = []string{"__pycache__"}
	}

	for _, line := range lines {
		if strings.TrimSpace(line) == pattern {
			yellow.Printf("'%s' is already in .mcignore\n", pattern)
			return nil
		}
	}

	lines = append(lines, pattern)
	content := strings.Join(lines, "\n")

	if err := os.WriteFile(mcignorePath, []byte(content), 0644); err != nil {
		return err
	}

	green.Printf("✓ Added '%s' to .mcignore\n", pattern)
	return nil
}

func Unignore(pattern string) error {
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	mcignorePath := filepath.Join(WorkspaceDir, ".mcignore")
	if _, err := os.Stat(mcignorePath); os.IsNotExist(err) {
		return fmt.Errorf("no .mcignore file found")
	}

	data, err := os.ReadFile(mcignorePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	var newLines []string
	removed := false

	for _, line := range lines {
		if strings.TrimSpace(line) != pattern {
			newLines = append(newLines, line)
		} else {
			removed = true
		}
	}

	if removed {
		content := strings.Join(newLines, "\n")
		if err := os.WriteFile(mcignorePath, []byte(content), 0644); err != nil {
			return err
		}
		green.Printf("✓ Removed '%s' from .mcignore\n", pattern)
	} else {
		yellow.Printf("Pattern '%s' not found in .mcignore\n", pattern)
	}

	return nil
}

func Lsignores() error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                      IGNORE PATTERNS                           ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

	mcignorePath := filepath.Join(WorkspaceDir, ".mcignore")
	if _, err := os.Stat(mcignorePath); os.IsNotExist(err) {
		yellow.Println("No .mcignore file found.\n")
		return nil
	}

	patterns, err := ReadMcignore()
	if err != nil {
		return err
	}

	if len(patterns) > 0 {
		for i, pattern := range patterns {
			green.Printf("  [%d] %s\n", i+1, pattern)
		}
		fmt.Println()
	} else {
		yellow.Println("No ignore patterns found.\n")
	}

	return nil
}

func ShouldIgnore(filePath string, ignorePatterns []string) bool {
	normalizedPath := filepath.Clean(filePath)

	if shouldSkipPath(normalizedPath) {
		return true
	}

	for _, pattern := range ignorePatterns {
		matched, _ := filepath.Match(pattern, normalizedPath)
		if matched {
			return true
		}

		matched, _ = filepath.Match(pattern, filepath.Base(normalizedPath))
		if matched {
			return true
		}

		if strings.Contains(normalizedPath, pattern) {
			return true
		}
	}

	return false
}

// isGlobPattern returns true if the string contains any glob metacharacters.
func isGlobPattern(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

// GatherFiles resolves a list of targets into a deduplicated, sorted list of file paths.
//
// Each target is handled as follows:
//
//  1. If the target contains a path separator or starts with "." or "..", it is
//     treated as a relative path from cwd. The file is included if it exists and
//     is not ignored.
//
//  2. If the target contains glob metacharacters (* ? [) and does NOT contain a
//     path separator, it is treated as a filename-only glob:
//     - Without -r: matched against files in cwd only.
//     - With -r: matched against filepath.Base of every file under cwd.
//
//  3. Otherwise the target is a plain filename:
//     - Without -r: looked up in cwd only.
//     - With -r: searched recursively by exact base name match.
//
// The shell may expand "*" into a list of filenames before the binary receives
// them. That is fine — each expanded filename falls into case 1 or 3 and is
// resolved correctly.
func GatherFiles(targets []string, recursive bool) ([]string, error) {
	ignorePatterns, err := ReadMcignore()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var matchingFiles []string

	addFile := func(path string) {
		clean := filepath.Clean(path)
		if seen[clean] {
			return
		}
		if ShouldIgnore(clean, ignorePatterns) {
			return
		}
		if strings.Contains(clean, WorkspaceDir) {
			return
		}
		seen[clean] = true
		matchingFiles = append(matchingFiles, clean)
	}

	for _, target := range targets {
		// Case 1: target contains a path separator or is a relative path indicator —
		// treat as a direct relative path.
		if strings.ContainsRune(target, os.PathSeparator) ||
			strings.HasPrefix(target, "./") ||
			strings.HasPrefix(target, "../") ||
			target == "." || target == ".." {

			info, err := os.Stat(target)
			if err != nil {
				// Not found — skip silently (could be a shell-expanded name that
				// doesn't exist, or a typo; caller sees 0 files for it).
				continue
			}
			if info.IsDir() {
				continue
			}
			addFile(target)
			continue
		}

		// Case 2 & 3: no path separator in target.
		if isGlobPattern(target) {
			// Glob pattern — match against base filenames.
			if !recursive {
				entries, err := os.ReadDir(".")
				if err != nil {
					return nil, err
				}
				for _, entry := range entries {
					if entry.IsDir() {
						continue
					}
					matched, _ := filepath.Match(target, entry.Name())
					if matched {
						addFile(entry.Name())
					}
				}
			} else {
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
					if ShouldIgnore(path, ignorePatterns) {
						if info.IsDir() {
							return filepath.SkipDir
						}
						return nil
					}
					if info.IsDir() {
						return nil
					}
					matched, _ := filepath.Match(target, filepath.Base(path))
					if matched {
						addFile(path)
					}
					return nil
				})
				if err != nil {
					return nil, err
				}
			}
		} else {
			// Plain filename — exact base name match.
			if !recursive {
				candidate := filepath.Join(".", target)
				info, err := os.Stat(candidate)
				if err == nil && !info.IsDir() {
					addFile(candidate)
				}
			} else {
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
					if ShouldIgnore(path, ignorePatterns) {
						if info.IsDir() {
							return filepath.SkipDir
						}
						return nil
					}
					if info.IsDir() {
						return nil
					}
					if filepath.Base(path) == target {
						addFile(path)
					}
					return nil
				})
				if err != nil {
					return nil, err
				}
			}
		}
	}

	sort.Strings(matchingFiles)
	return matchingFiles, nil
}
