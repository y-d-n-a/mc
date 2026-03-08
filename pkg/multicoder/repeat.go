
package multicoder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
)

const LastCallFile = ".last_call"

type LastCall struct {
	Command          string   `json:"command"`
	LLMCount         int      `json:"llm_count"`
	Targets          []string `json:"targets"`
	Recursive        bool     `json:"recursive"`
	UserInstructions string   `json:"user_instructions"`
}

func SaveLastCall(llmCount int, targets []string, recursive bool, userInstructions string) error {
	lastCall := LastCall{
		Command:          "get",
		LLMCount:         llmCount,
		Targets:          targets,
		Recursive:        recursive,
		UserInstructions: userInstructions,
	}

	data, err := json.MarshalIndent(lastCall, "", "  ")
	if err != nil {
		return err
	}

	lastCallPath := filepath.Join(WorkspaceDir, LastCallFile)
	if err := os.MkdirAll(WorkspaceDir, 0755); err != nil {
		return err
	}

	return os.WriteFile(lastCallPath, data, 0644)
}

func LoadLastCall() (*LastCall, error) {
	lastCallPath := filepath.Join(WorkspaceDir, LastCallFile)

	data, err := os.ReadFile(lastCallPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no previous call found. Use 'mc get' first")
		}
		return nil, err
	}

	var lastCall LastCall
	if err := json.Unmarshal(data, &lastCall); err != nil {
		return nil, err
	}

	return &lastCall, nil
}

func HandleRepeat(repeatCount int) error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                    MULTICODER REPEAT                           ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

	yellow.Println("→ Loading last call...")
	lastCall, err := LoadLastCall()
	if err != nil {
		return err
	}

	green.Printf("  ✓ Last call loaded\n")
	green.Printf("    Command: %s\n", lastCall.Command)
	green.Printf("    LLM Count: %d\n", lastCall.LLMCount)
	green.Printf("    Targets: %v\n", lastCall.Targets)
	green.Printf("    Recursive: %v\n", lastCall.Recursive)
	if lastCall.UserInstructions != "" {
		green.Printf("    Instructions: %s\n", lastCall.UserInstructions)
	}
	fmt.Println()

	cyan.Printf("→ Repeating call %d time(s)...\n\n", repeatCount)

	for i := 0; i < repeatCount; i++ {
		if repeatCount > 1 {
			yellow.Printf("═══ Iteration %d of %d ═══\n\n", i+1, repeatCount)
		}

		if err := HandleGet(lastCall.LLMCount, lastCall.Targets, lastCall.Recursive, lastCall.UserInstructions); err != nil {
			return fmt.Errorf("repeat iteration %d failed: %v", i+1, err)
		}

		if i < repeatCount-1 {
			fmt.Println()
		}
	}

	fmt.Println()
	green.Printf("✓ Repeat completed: %d iteration(s)\n\n", repeatCount)

	return nil
}
