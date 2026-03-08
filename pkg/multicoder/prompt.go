
package multicoder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
	"mc/pkg/shared"
)

const (
	ActivePromptFile = ".prompt"
)

func getSysPromptsDir() (string, error) {
	projectRoot, err := shared.GetProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, ".sys_prompts"), nil
}

func getActivePromptFile() string {
	return filepath.Join(WorkspaceDir, ActivePromptFile)
}

func ensureSysPromptsDir() error {
	sysPromptsDir, err := getSysPromptsDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(sysPromptsDir, 0755)
}

func getPromptPath(name string) (string, error) {
	sysPromptsDir, err := getSysPromptsDir()
	if err != nil {
		return "", err
	}
	safeName := strings.ReplaceAll(name, "/", "_")
	safeName = strings.ReplaceAll(safeName, "..", "_")
	return filepath.Join(sysPromptsDir, safeName+".txt"), nil
}

func HandlePromptAdd(name string) error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                    ADD SYSTEM PROMPT                           ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

	if err := ensureSysPromptsDir(); err != nil {
		return fmt.Errorf("failed to create prompts directory: %v", err)
	}

	promptPath, err := getPromptPath(name)
	if err != nil {
		return err
	}

	if _, err := os.Stat(promptPath); err == nil {
		red.Printf("Prompt '%s' already exists\n", name)
		yellow.Println("Use 'mc prompt update' to modify it\n")
		return fmt.Errorf("prompt already exists")
	}

	yellow.Println("→ Opening editor for new prompt...")
	content, err := openEditorForPrompt("")
	if err != nil {
		return err
	}

	if err := os.WriteFile(promptPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to save prompt: %v", err)
	}

	green.Printf("✓ Prompt '%s' created\n", name)
	green.Printf("  Path: %s\n", promptPath)
	yellow.Println("\nUse 'mc prompt switch' to activate it\n")

	return nil
}

func HandlePromptDelete(name string) error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                   DELETE SYSTEM PROMPT                         ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

	promptPath, err := getPromptPath(name)
	if err != nil {
		return err
	}

	if _, err := os.Stat(promptPath); os.IsNotExist(err) {
		red.Printf("Prompt '%s' does not exist\n\n", name)
		return fmt.Errorf("prompt not found")
	}

	activePrompt, _ := getActivePrompt()
	if activePrompt == name {
		yellow.Printf("⚠ Warning: '%s' is currently active\n", name)
	}

	fmt.Print("Delete this prompt? (y/N): ")
	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))

	if response != "y" && response != "yes" {
		yellow.Println("Operation cancelled\n")
		return nil
	}

	if err := os.Remove(promptPath); err != nil {
		return fmt.Errorf("failed to delete prompt: %v", err)
	}

	if activePrompt == name {
		os.Remove(getActivePromptFile())
		yellow.Println("→ Active prompt cleared (no system prompt will be used)")
	}

	green.Printf("✓ Prompt '%s' deleted\n\n", name)
	return nil
}

func HandlePromptUpdate(name string) error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                   UPDATE SYSTEM PROMPT                         ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

	promptPath, err := getPromptPath(name)
	if err != nil {
		return err
	}

	if _, err := os.Stat(promptPath); os.IsNotExist(err) {
		red.Printf("Prompt '%s' does not exist\n", name)
		yellow.Println("Use 'mc prompt add' to create it\n")
		return fmt.Errorf("prompt not found")
	}

	currentContent, err := os.ReadFile(promptPath)
	if err != nil {
		return fmt.Errorf("failed to read prompt: %v", err)
	}

	yellow.Println("→ Opening editor to update prompt...")
	newContent, err := openEditorForPrompt(string(currentContent))
	if err != nil {
		return err
	}

	if err := os.WriteFile(promptPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to save prompt: %v", err)
	}

	green.Printf("✓ Prompt '%s' updated\n", name)
	green.Printf("  Path: %s\n\n", promptPath)

	return nil
}

func HandlePromptSwitch(name string) error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                   SWITCH SYSTEM PROMPT                         ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

	if name == "" {
		return HandlePromptSwitchInteractive()
	}

	if name == "null" {
		if err := os.MkdirAll(WorkspaceDir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(getActivePromptFile(), []byte("null"), 0644); err != nil {
			return fmt.Errorf("failed to set null prompt: %v", err)
		}
		green.Println("✓ No system prompt will be used\n")
		return nil
	}

	promptPath, err := getPromptPath(name)
	if err != nil {
		return err
	}

	if _, err := os.Stat(promptPath); os.IsNotExist(err) {
		red.Printf("Prompt '%s' does not exist\n", name)
		yellow.Println("Use 'mc prompt list' to see available prompts\n")
		return fmt.Errorf("prompt not found")
	}

	if err := os.MkdirAll(WorkspaceDir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(getActivePromptFile(), []byte(name), 0644); err != nil {
		return fmt.Errorf("failed to set active prompt: %v", err)
	}

	green.Printf("✓ Switched to prompt: '%s'\n", name)
	green.Printf("  Path: %s\n\n", promptPath)

	return nil
}

func HandlePromptSwitchInteractive() error {
	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                   SELECT SYSTEM PROMPT                         ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

	prompts, err := getAvailablePrompts()
	if err != nil {
		return err
	}

	if len(prompts) == 0 {
		yellow.Println("No custom prompts found")
		yellow.Println("Use 'mc prompt add <name>' to create one\n")
		return nil
	}

	options := append([]string{"null (no system prompt)"}, prompts...)

	yellow.Println("Select a prompt:\n")
	choice, err := readArrowKeySelection(options, "  ")
	if err != nil {
		return err
	}

	if choice == 0 {
		return HandlePromptSwitch("null")
	} else {
		return HandlePromptSwitch(prompts[choice-1])
	}
}

func HandlePromptList() error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                    SYSTEM PROMPTS                              ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

	activePrompt, _ := getActivePrompt()

	yellow.Println("Active prompt:")
	if activePrompt == "" {
		green.Println("  → null (no system prompt)\n")
	} else if activePrompt == "null" {
		green.Println("  → null (no system prompt)\n")
	} else {
		green.Printf("  → %s\n\n", activePrompt)
	}

	prompts, err := getAvailablePrompts()
	if err != nil {
		return err
	}

	if len(prompts) == 0 {
		yellow.Println("No custom prompts found")
		yellow.Println("Use 'mc prompt add <name>' to create one\n")
		return nil
	}

	yellow.Println("Available prompts:")
	for _, name := range prompts {
		if name == activePrompt {
			green.Printf("  → %s (active)\n", name)
		} else {
			fmt.Printf("    %s\n", name)
		}
	}
	fmt.Println()

	return nil
}

func getActivePrompt() (string, error) {
	data, err := os.ReadFile(getActivePromptFile())
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func getAvailablePrompts() ([]string, error) {
	sysPromptsDir, err := getSysPromptsDir()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(sysPromptsDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(sysPromptsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompts directory: %v", err)
	}

	var prompts []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
			name := strings.TrimSuffix(entry.Name(), ".txt")
			prompts = append(prompts, name)
		}
	}

	sort.Strings(prompts)
	return prompts, nil
}

func EnsurePromptSet() error {
	activePrompt, _ := getActivePrompt()
	if activePrompt != "" {
		return nil
	}

	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                  NO SYSTEM PROMPT SET                          ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

	yellow.Println("No system prompt is currently set for this repository.")
	fmt.Print("\nWould you like to set one? (y/N): ")

	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))

	if response != "y" && response != "yes" {
		if err := os.MkdirAll(WorkspaceDir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(getActivePromptFile(), []byte("null"), 0644); err != nil {
			return fmt.Errorf("failed to set null prompt: %v", err)
		}
		yellow.Println("\n✓ Set to 'null' - no system prompt will be used\n")
		return nil
	}

	return HandlePromptSwitchInteractive()
}

func GetSystemPrompt() string {
	activePrompt, err := getActivePrompt()
	if err != nil || activePrompt == "" {
		return ""
	}

	if activePrompt == "null" {
		return ""
	}

	promptPath, err := getPromptPath(activePrompt)
	if err != nil {
		return ""
	}

	content, err := os.ReadFile(promptPath)
	if err != nil {
		return ""
	}

	return string(content)
}

func openEditorForPrompt(initialContent string) (string, error) {
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)

	if err := os.MkdirAll(WorkspaceDir, 0755); err != nil {
		return "", err
	}

	tempFile := filepath.Join(WorkspaceDir, "temp_prompt.txt")
	defer os.Remove(tempFile)

	if err := os.WriteFile(tempFile, []byte(initialContent), 0644); err != nil {
		return "", err
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nvim"
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

	if content == "" {
		red.Println("\n✗ No content provided. Operation cancelled.\n")
		return "", fmt.Errorf("no content provided")
	}

	yellow.Printf("  ✓ Prompt captured (%d bytes)\n\n", len(content))

	return content, nil
}
