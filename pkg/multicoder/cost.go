
package multicoder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
)

const CostFile = ".cost"

type CostEntry struct {
	Timestamp    string  `json:"timestamp"`
	Model        string  `json:"model"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	InputCost    float64 `json:"input_cost"`
	OutputCost   float64 `json:"output_cost"`
	TotalCost    float64 `json:"total_cost"`
}

type ProjectCost struct {
	TotalCost   float64     `json:"total_cost"`
	TotalCalls  int         `json:"total_calls"`
	CostEntries []CostEntry `json:"cost_entries"`
}

func getCostFilePath() string {
	return filepath.Join(WorkspaceDir, CostFile)
}

func LoadProjectCost() (*ProjectCost, error) {
	costPath := getCostFilePath()

	if _, err := os.Stat(costPath); os.IsNotExist(err) {
		return &ProjectCost{
			TotalCost:   0,
			TotalCalls:  0,
			CostEntries: make([]CostEntry, 0),
		}, nil
	}

	data, err := os.ReadFile(costPath)
	if err != nil {
		return nil, err
	}

	var projectCost ProjectCost
	if err := json.Unmarshal(data, &projectCost); err != nil {
		return nil, err
	}

	return &projectCost, nil
}

func SaveProjectCost(projectCost *ProjectCost) error {
	if err := os.MkdirAll(WorkspaceDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(projectCost, "", "  ")
	if err != nil {
		return err
	}

	costPath := getCostFilePath()
	return os.WriteFile(costPath, data, 0644)
}

func AddCostEntry(timestamp, model string, inputTokens, outputTokens int, inputCost, outputCost, totalCost float64) error {
	projectCost, err := LoadProjectCost()
	if err != nil {
		return err
	}

	entry := CostEntry{
		Timestamp:    timestamp,
		Model:        model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		InputCost:    inputCost,
		OutputCost:   outputCost,
		TotalCost:    totalCost,
	}

	projectCost.CostEntries = append(projectCost.CostEntries, entry)
	projectCost.TotalCost += totalCost
	projectCost.TotalCalls++

	return SaveProjectCost(projectCost)
}

func ShowProjectCost() error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)

	projectCost, err := LoadProjectCost()
	if err != nil {
		return err
	}

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                     PROJECT COST SUMMARY                       ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝")

	fmt.Printf("\n%-20s %s\n", cyan.Sprint("Total API Calls:"), green.Sprintf("%d", projectCost.TotalCalls))
	
	var totalColor *color.Color
	if projectCost.TotalCost > 1.0 {
		totalColor = red
	} else if projectCost.TotalCost > 0.1 {
		totalColor = yellow
	} else {
		totalColor = green
	}
	
	fmt.Printf("%-20s %s\n\n", cyan.Sprint("Total Cost:"), totalColor.Sprintf("$%.6f", projectCost.TotalCost))

	if len(projectCost.CostEntries) > 0 {
		cyan.Println("Recent API Calls:")
		fmt.Println(color.HiBlackString("─────────────────────────────────────────────────────────────────"))

		startIdx := 0
		if len(projectCost.CostEntries) > 10 {
			startIdx = len(projectCost.CostEntries) - 10
		}

		for i := startIdx; i < len(projectCost.CostEntries); i++ {
			entry := projectCost.CostEntries[i]
			fmt.Printf("%s  %s  %s\n",
				color.HiBlackString(entry.Timestamp),
				yellow.Sprintf("%-40s", entry.Model),
				green.Sprintf("$%.6f", entry.TotalCost))
		}

		if len(projectCost.CostEntries) > 10 {
			fmt.Println(color.HiBlackString("... and", len(projectCost.CostEntries)-10, "more"))
		}
	}

	fmt.Println()
	return nil
}

func ClearProjectCost() error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)

	costPath := getCostFilePath()
	
	if _, err := os.Stat(costPath); os.IsNotExist(err) {
		cyan.Println("\nNo cost data to clear\n")
		return nil
	}

	if err := os.Remove(costPath); err != nil {
		return err
	}

	green.Println("\n✓ Project cost data cleared\n")
	return nil
}
