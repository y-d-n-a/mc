
package multicoder

import (
        "fmt"
        "os"
        "path/filepath"
        "regexp"

        "github.com/fatih/color"
)

func HandleWrite(m int) error {
        cyan := color.New(color.FgCyan, color.Bold)
        green := color.New(color.FgGreen)
        yellow := color.New(color.FgYellow)
        red := color.New(color.FgRed)

        cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
        cyan.Println("║                    MULTICODER WRITE STARTED                    ║")
        cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

        cyan.Println("→ Locating version folder...")
        versionFolder, err := GetLatestVersionFolder()
        if err != nil {
                return err
        }
        green.Printf("  ✓ Using: %s\n\n", versionFolder)

        cyan.Printf("→ Reading response %d...\n", m)
        responseFile := filepath.Join(versionFolder, "responses", fmt.Sprintf("response%d.txt", m))
        responseContent, err := ReadResponseFile(responseFile)
        if err != nil {
                return err
        }
        green.Println("  ✓ Response loaded\n")

        cyan.Println("→ Parsing files from response...")
        filePattern := regexp.MustCompile(`(?s)<file path="([^"]+)">(.+?)</file>`)
        matches := filePattern.FindAllStringSubmatch(responseContent, -1)

        if len(matches) == 0 {
                return fmt.Errorf("no files found in response")
        }
        green.Printf("  ✓ Found %d file(s) to write\n\n", len(matches))

        cyan.Println("→ Creating backup of current state...")
        if err := BackupCurrentState(); err != nil {
                return err
        }
        green.Println("  ✓ Backup created\n")

        cyan.Println("→ Writing files...\n")
        for i, match := range matches {
                filePath := match[1]
                fileContent := match[2]

                if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
                        red.Printf("  ✗ [%d/%d] Failed to create directory for %s: %v\n", i+1, len(matches), filePath, err)
                        continue
                }

                if err := os.WriteFile(filePath, []byte(fileContent), 0644); err != nil {
                        red.Printf("  ✗ [%d/%d] Failed to write %s: %v\n", i+1, len(matches), filePath, err)
                        continue
                }

                green.Printf("  ✓ [%d/%d] %s\n", i+1, len(matches), filePath)
        }

        fmt.Println()
        green.Printf("✓ Write operation completed: %d file(s) written\n", len(matches))
        yellow.Println("→ Use 'mc undo' to revert if needed\n")

        return nil
}