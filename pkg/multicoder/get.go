
package multicoder

import (
        "fmt"
        "os"
        "path/filepath"
        "strings"
        "sync"
        "time"

        "github.com/fatih/color"
        "mc/pkg/ai"
        "mc/pkg/shared"
)

func HandleGet(llmCount int, targets []string, recursive bool, userInstructions string) error {
        cyan := color.New(color.FgCyan, color.Bold)
        green := color.New(color.FgGreen)
        yellow := color.New(color.FgYellow)
        red := color.New(color.FgRed)

        cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
        cyan.Println("║                    MULTICODER GET STARTED                      ║")
        cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

        if err := EnsurePromptSet(); err != nil {
                return err
        }

        if userInstructions == "" {
                yellow.Println("→ Opening editor for instructions...")
                instructions, err := GetUserInstructionsFromEditor()
                if err != nil {
                        return err
                }
                userInstructions = instructions
        }

        yellow.Println("→ Saving call for repeat...")
        if err := SaveLastCall(llmCount, targets, recursive, userInstructions); err != nil {
                yellow.Printf("  ⚠ Warning: Could not save last call: %v\n", err)
        } else {
                green.Println("  ✓ Call saved for repeat\n")
        }

        cyan.Println("→ Creating version folder...")
        versionFolder, err := CreateVersionFolder()
        if err != nil {
                return err
        }
        green.Printf("  ✓ Version folder: %s\n\n", versionFolder)

        backupFolder := filepath.Join(versionFolder, "backup")
        responseFolder := filepath.Join(versionFolder, "responses")

        if err := os.MkdirAll(backupFolder, 0755); err != nil {
                return err
        }
        if err := os.MkdirAll(responseFolder, 0755); err != nil {
                return err
        }

        cyan.Println("→ Gathering files...")
        files, err := GatherFiles(targets, recursive)
        if err != nil {
                return err
        }
        green.Printf("  ✓ Found %d files\n\n", len(files))

        cyan.Println("→ Backing up files...")
        backedUp := 0
        skippedCount := 0
        for _, file := range files {
                if shouldSkipPath(file) {
                        skippedCount++
                        continue
                }

                absFile, err := filepath.Abs(file)
                if err != nil {
                        continue
                }

                cwd, err := os.Getwd()
                if err != nil {
                        continue
                }

                var backupFilePath string
                if strings.HasPrefix(absFile, cwd+string(os.PathSeparator)) || absFile == cwd {
                        relPath, _ := filepath.Rel(cwd, absFile)
                        backupFilePath = filepath.Join(backupFolder, relPath)
                } else {
                        relPath, _ := filepath.Rel("/", absFile)
                        backupFilePath = filepath.Join(backupFolder, "external", relPath)
                }

                if err := os.MkdirAll(filepath.Dir(backupFilePath), 0755); err != nil {
                        continue
                }

                data, err := os.ReadFile(file)
                if err != nil {
                        continue
                }
                os.WriteFile(backupFilePath, data, 0644)
                backedUp++
        }
        green.Printf("  ✓ Backed up %d files\n", backedUp)
        if skippedCount > 0 {
                yellow.Printf("  ✓ Skipped %d version control file(s)\n", skippedCount)
        }
        fmt.Println()

        cyan.Println("→ Building prompt...")
        systemPrompt := GetSystemPrompt()
        prompt := ""
        if systemPrompt != "" {
                prompt = systemPrompt + "\n"
        }
        prompt += fmt.Sprintf("<user instructions>\n%s\n</user instructions>\n", userInstructions)
        prompt += "<project files>\n"

        for _, file := range files {
                content, err := os.ReadFile(file)
                if err != nil {
                        red.Printf("  ✗ Error reading file %s: %v\n", file, err)
                        yellow.Printf("  → Suggestion: Run 'mc ignore %s' to ignore this file\n", file)
                        continue
                }
                prompt += fmt.Sprintf("<file path=\"%s\">%s</file>\n", file, string(content))
        }
        prompt += "</project files>\n"

        projectRoot, err := shared.GetProjectRoot()
        if err != nil {
                return fmt.Errorf("failed to get project root: %v", err)
        }

        envPath := filepath.Join(projectRoot, ".env")
        envData, err := os.ReadFile(envPath)
        if err != nil {
                return fmt.Errorf("failed to read .env from %s: %v", envPath, err)
        }

        model := ""
        for _, line := range strings.Split(string(envData), "\n") {
                if strings.HasPrefix(line, "AI_TOOLS_MODEL=") {
                        model = strings.TrimPrefix(line, "AI_TOOLS_MODEL=")
                        model = strings.TrimSpace(model)
                        break
                }
        }

        if model == "" {
                model = "anthropic/claude-sonnet-4"
        }
        green.Printf("  ✓ Using model: %s\n\n", model)

        modelInterface, err := ai.NewModelInterface("", "")
        if err != nil {
                return err
        }

        cyan.Printf("→ Sending to %d LLM instance(s) with max_tokens=%d...\n\n", llmCount, modelInterface.MaxTokens)

        var wg sync.WaitGroup
        responseChan := make(chan struct {
                index    int
                response string
                err      error
        }, llmCount)

        for i := 0; i < llmCount; i++ {
                wg.Add(1)
                yellow.Printf("  ⟳ LLM %d: Processing...\n", i)
                go func(index int) {
                        defer wg.Done()
                        response, err := modelInterface.SendToAI(prompt, model, 0, 0.7, "", nil)
                        responseChan <- struct {
                                index    int
                                response string
                                err      error
                        }{index, response, err}
                }(i)
        }

        go func() {
                wg.Wait()
                close(responseChan)
        }()

        fmt.Println()
        successCount := 0
        for result := range responseChan {
                if result.err != nil {
                        red.Printf("  ✗ LLM %d failed: %v\n", result.index, result.err)
                        continue
                }
                if err := SaveResponse(responseFolder, result.index, result.response); err != nil {
                        red.Printf("  ✗ Error saving response %d: %v\n", result.index, err)
                        continue
                }
                green.Printf("  ✓ LLM %d: Response saved\n", result.index)
                successCount++
        }

        fmt.Println()
        if successCount == llmCount {
                green.Printf("✓ All %d responses completed successfully\n", llmCount)
        } else {
                yellow.Printf("⚠ %d/%d responses completed\n", successCount, llmCount)
        }

        modelInterface.CostTracker.ShowCostData()

        timestamp := time.Now().Format("2006-01-02 15:04:05")
        for _, costData := range modelInterface.CostTracker.CostData {
                if err := AddCostEntry(timestamp, costData.Model, 0, 0, costData.InputCost, costData.OutputCost, costData.TotalCost); err != nil {
                        yellow.Printf("⚠ Warning: Could not save cost data: %v\n", err)
                }
        }

        return nil
}