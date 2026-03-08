
package multicoder

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
)

func ListResponses() error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                      AVAILABLE RESPONSES                       ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

	versionFolder, err := GetLatestVersionFolder()
	if err != nil {
		return err
	}

	yellow.Printf("Version: %s\n\n", filepath.Base(versionFolder))

	responseFolder := filepath.Join(versionFolder, "responses")
	entries, err := os.ReadDir(responseFolder)
	if err != nil {
		return err
	}

	var responseFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "response") && strings.HasSuffix(entry.Name(), ".txt") {
			responseFiles = append(responseFiles, entry.Name())
		}
	}

	sort.Strings(responseFiles)

	if len(responseFiles) == 0 {
		yellow.Println("No responses found")
		return nil
	}

	for i, file := range responseFiles {
		filePath := filepath.Join(responseFolder, file)
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		size := info.Size()
		var sizeStr string
		if size < 1024 {
			sizeStr = fmt.Sprintf("%d B", size)
		} else if size < 1024*1024 {
			sizeStr = fmt.Sprintf("%.1f KB", float64(size)/1024)
		} else {
			sizeStr = fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
		}

		green.Printf("  [%d] %s ", i, file)
		color.HiBlack("(%s)", sizeStr)
		fmt.Println()
	}

	fmt.Println()
	return nil
}

func OpenResponse(responseIndex int) error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                       RESPONSE CONTENTS                        ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

	versionFolder, err := GetLatestVersionFolder()
	if err != nil {
		return err
	}

	responseFolder := filepath.Join(versionFolder, "responses")
	responseFile := filepath.Join(responseFolder, fmt.Sprintf("response%d.txt", responseIndex))

	content, err := os.ReadFile(responseFile)
	if err != nil {
		return fmt.Errorf("failed to read response %d: %v", responseIndex, err)
	}

	fileName := fmt.Sprintf("response%d.txt", responseIndex)
	cyan.Printf("┌─ %s ", fileName)
	cyan.Println(strings.Repeat("─", 60-len(fileName)))
	green.Println(string(content))
	cyan.Printf("└─ End of %s ", fileName)
	cyan.Println(strings.Repeat("─", 55-len(fileName)))

	fmt.Println()
	return nil
}

func OpenResponses() error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                       RESPONSE CONTENTS                        ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

	versionFolder, err := GetLatestVersionFolder()
	if err != nil {
		return err
	}

	responseFolder := filepath.Join(versionFolder, "responses")
	entries, err := os.ReadDir(responseFolder)
	if err != nil {
		return err
	}

	var responseFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "response") && strings.HasSuffix(entry.Name(), ".txt") {
			responseFiles = append(responseFiles, entry.Name())
		}
	}

	sort.Strings(responseFiles)

	if len(responseFiles) == 0 {
		yellow.Println("No responses found")
		return nil
	}

	for i, file := range responseFiles {
		filePath := filepath.Join(responseFolder, file)
		content, err := os.ReadFile(filePath)
		if err != nil {
			red.Printf("Error reading %s: %v\n", file, err)
			continue
		}

		if i > 0 {
			fmt.Println()
		}

		cyan.Printf("┌─ %s ", file)
		cyan.Println(strings.Repeat("─", 60-len(file)))
		green.Println(string(content))
		cyan.Printf("└─ End of %s ", file)
		cyan.Println(strings.Repeat("─", 55-len(file)))
	}

	fmt.Println()
	return nil
}
