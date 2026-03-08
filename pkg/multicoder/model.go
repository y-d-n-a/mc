
package multicoder

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
	"mc/pkg/ai"
	"mc/pkg/shared"
)

func HandleModel(subcommand string, modelID string) error {
	cyan := color.New(color.FgCyan, color.Bold)

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                       MODEL MANAGEMENT                         ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

	switch subcommand {
	case "add":
		return handleModelAdd()
	case "remove":
		return handleModelRemove()
	case "":
		return handleModelSelect()
	default:
		return fmt.Errorf("unknown subcommand: %s (use 'add' or 'remove')", subcommand)
	}
}

// modelsFilePath returns the absolute path to models.json in the project root.
func modelsFilePath() (string, error) {
	projectRoot, err := shared.GetProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, "models.json"), nil
}

// loadSavedModels reads models.json and returns the parsed slice.
// Returns an empty slice (not an error) when the file does not yet exist.
func loadSavedModels(modelsPath string) ([]ai.OpenRouterModel, error) {
	var models []ai.OpenRouterModel
	data, err := os.ReadFile(modelsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return models, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, &models); err != nil {
		return nil, err
	}
	return models, nil
}

// saveModels serialises models and writes them to modelsPath.
func saveModels(modelsPath string, models []ai.OpenRouterModel) error {
	sort.Slice(models, func(i, j int) bool {
		return models[i].ID < models[j].ID
	})
	data, err := json.MarshalIndent(models, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(modelsPath, data, 0644)
}

// handleModelAdd presents a two-option menu: add from OpenRouter API or add a
// local model manually.
func handleModelAdd() error {
	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)

	if err := shared.LoadEnvFile(); err != nil {
		return fmt.Errorf("failed to load .env file: %v", err)
	}

	modelsPath, err := modelsFilePath()
	if err != nil {
		return err
	}

	savedModels, err := loadSavedModels(modelsPath)
	if err != nil {
		return err
	}

	cyan.Println("Choose how to add a model:\n")
	yellow.Println("  [1] Add from OpenRouter API (cloud models)")
	yellow.Println("  [2] Add local model manually  (ollama/...)")
	fmt.Println()

	choice, err := readArrowKeySelection([]string{
		"Add from OpenRouter API (cloud models)",
		"Add local model manually (ollama/...)",
	}, "  ")
	if err != nil {
		return err
	}

	switch choice {
	case 0:
		return handleModelAddFromAPI(savedModels, modelsPath)
	case 1:
		return handleModelAddLocal(savedModels, modelsPath)
	}
	return nil
}

// handleModelAddFromAPI fetches all models from OpenRouter, filters out ones
// already saved, and lets the user multi-select from the remainder.
func handleModelAddFromAPI(savedModels []ai.OpenRouterModel, modelsPath string) error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	yellow.Println("→ Fetching all models from OpenRouter...")
	allModels, err := ai.FetchOpenRouterModels()
	if err != nil {
		return fmt.Errorf("failed to fetch models: %v", err)
	}
	green.Printf("  ✓ Fetched %d models from API\n\n", len(allModels))

	savedIDs := make(map[string]bool, len(savedModels))
	for _, m := range savedModels {
		savedIDs[m.ID] = true
	}

	var available []ai.OpenRouterModel
	for _, m := range allModels {
		if !savedIDs[m.ID] {
			available = append(available, m)
		}
	}

	if len(available) == 0 {
		yellow.Println("All OpenRouter models are already in your saved list.\n")
		return nil
	}

	sort.Slice(available, func(i, j int) bool {
		return available[i].ID < available[j].ID
	})

	cyan.Printf("\n╔════════════════════════════════════════════════════════════════╗\n")
	cyan.Printf("║  %d models available to add                                    ║\n", len(available))
	cyan.Printf("╚════════════════════════════════════════════════════════════════╝\n\n")
	yellow.Println("Use SPACE to select/deselect, ENTER to confirm, ESC to cancel\n")

	selectedIndices, err := readMultiSelectArrowKey(available)
	if err != nil {
		return err
	}

	if len(selectedIndices) == 0 {
		yellow.Println("\nNo models selected.\n")
		return nil
	}

	cyan.Printf("\n→ Adding %d model(s)...\n\n", len(selectedIndices))
	for _, idx := range selectedIndices {
		savedModels = append(savedModels, available[idx])
		green.Printf("  ✓ Added: %s\n", available[idx].ID)
	}

	if err := saveModels(modelsPath, savedModels); err != nil {
		return err
	}

	fmt.Println()
	green.Printf("✓ Successfully added %d model(s)\n\n", len(selectedIndices))
	return nil
}

// handleModelAddLocal prompts for a model ID with the "ollama/" prefix and
// appends a local model entry to models.json.
func handleModelAddLocal(savedModels []ai.OpenRouterModel, modelsPath string) error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)

	cyan.Println("\nEnter the local model ID (must start with 'ollama/'):")
	yellow.Println("Examples:  ollama/qwen3.5   ollama/llama3   ollama/mistral\n")
	fmt.Print("Model ID: ")

	var modelID string
	fmt.Scanln(&modelID)
	modelID = strings.TrimSpace(modelID)

	if modelID == "" {
		red.Println("No model ID provided. Operation cancelled.\n")
		return nil
	}

	if !strings.HasPrefix(modelID, "ollama/") {
		red.Printf("Local models must start with 'ollama/'. Got: %s\n\n", modelID)
		return fmt.Errorf("invalid local model ID: %s — must start with 'ollama/'", modelID)
	}

	// Reject bare "ollama/" with nothing after the slash.
	if strings.TrimPrefix(modelID, "ollama/") == "" {
		red.Println("Model name after 'ollama/' cannot be empty.\n")
		return fmt.Errorf("empty model name in ID: %s", modelID)
	}

	for _, m := range savedModels {
		if m.ID == modelID {
			yellow.Printf("Model '%s' is already in models.json\n\n", modelID)
			return nil
		}
	}

	newModel := ai.NewLocalModel(modelID)
	savedModels = append(savedModels, newModel)

	if err := saveModels(modelsPath, savedModels); err != nil {
		return err
	}

	green.Printf("✓ Added local model: %s\n\n", modelID)
	return nil
}

// readMultiSelectArrowKey renders an interactive multi-select list and returns
// the indices of selected items.
func readMultiSelectArrowKey(models []ai.OpenRouterModel) ([]int, error) {
	execCmd := func(cmd string, args ...string) error {
		c := exec.Command(cmd, args...)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	}

	if err := execCmd("stty", "-echo", "-icanon"); err != nil {
		return nil, err
	}
	defer execCmd("stty", "echo", "icanon")

	currentSelection := 0
	viewOffset := 0
	selected := make(map[int]bool)

	green := color.New(color.FgGreen)
	white := color.New(color.FgWhite)
	yellow := color.New(color.FgYellow)
	cyan := color.New(color.FgCyan, color.Bold)

	termHeight := getTerminalHeight()
	visibleLines := termHeight - 8

	printOptions := func() {
		fmt.Print("\033[H\033[2J")

		cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
		cyan.Println("║                  SELECT MODELS TO ADD                          ║")
		cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")
		yellow.Println("Use SPACE to select/deselect, ENTER to confirm, ESC to cancel\n")

		endIdx := viewOffset + visibleLines
		if endIdx > len(models) {
			endIdx = len(models)
		}

		for i := viewOffset; i < endIdx; i++ {
			model := models[i]
			checkbox := "[ ]"
			if selected[i] {
				checkbox = "[✓]"
			}

			localTag := ""
			if model.IsLocal || ai.IsLocalModel(model.ID) {
				localTag = " [local]"
			}
			displayText := fmt.Sprintf("%s %s%s", checkbox, model.ID, localTag)

			if i == currentSelection {
				green.Printf("  → %s\n", displayText)
			} else if selected[i] {
				cyan.Printf("    %s\n", displayText)
			} else {
				white.Printf("    %s\n", displayText)
			}
		}

		fmt.Println()
		if len(selected) > 0 {
			green.Printf("  Selected: %d model(s)\n", len(selected))
		}
		yellow.Printf("  Viewing: %d-%d of %d\n", viewOffset+1, endIdx, len(models))
	}

	printOptions()

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return nil, err
		}

		if n == 1 {
			switch buf[0] {
			case 13, 10: // Enter
				fmt.Print("\033[H\033[2J")
				var result []int
				for idx := range selected {
					result = append(result, idx)
				}
				return result, nil
			case 32: // Space
				if selected[currentSelection] {
					delete(selected, currentSelection)
				} else {
					selected[currentSelection] = true
				}
				printOptions()
			case 27: // Escape
				fmt.Print("\033[H\033[2J")
				return []int{}, nil
			}
		}

		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 65: // Up arrow
				if currentSelection > 0 {
					currentSelection--
					if currentSelection < viewOffset {
						viewOffset = currentSelection
					}
					printOptions()
				}
			case 66: // Down arrow
				if currentSelection < len(models)-1 {
					currentSelection++
					if currentSelection >= viewOffset+visibleLines {
						viewOffset = currentSelection - visibleLines + 1
					}
					printOptions()
				}
			}
		}
	}
}

func handleModelRemove() error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	if err := shared.LoadEnvFile(); err != nil {
		return fmt.Errorf("failed to load .env file: %v", err)
	}

	modelsPath, err := modelsFilePath()
	if err != nil {
		return err
	}

	savedModels, err := loadSavedModels(modelsPath)
	if err != nil {
		return err
	}

	if len(savedModels) == 0 {
		yellow.Println("No saved models to remove.\n")
		return nil
	}

	sort.Slice(savedModels, func(i, j int) bool {
		return savedModels[i].ID < savedModels[j].ID
	})

	cyan.Printf("\n╔════════════════════════════════════════════════════════════════╗\n")
	cyan.Printf("║  %d models available to remove                                 ║\n", len(savedModels))
	cyan.Printf("╚════════════════════════════════════════════════════════════════╝\n\n")
	yellow.Println("Use SPACE to select/deselect, ENTER to confirm, ESC to cancel\n")

	selectedIndices, err := readMultiSelectArrowKey(savedModels)
	if err != nil {
		return err
	}

	if len(selectedIndices) == 0 {
		yellow.Println("\nNo models selected.\n")
		return nil
	}

	cyan.Printf("\n→ Removing %d model(s)...\n\n", len(selectedIndices))

	removeSet := make(map[int]bool, len(selectedIndices))
	for _, idx := range selectedIndices {
		removeSet[idx] = true
		green.Printf("  ✓ Removed: %s\n", savedModels[idx].ID)
	}

	var updated []ai.OpenRouterModel
	for i, m := range savedModels {
		if !removeSet[i] {
			updated = append(updated, m)
		}
	}

	if err := saveModels(modelsPath, updated); err != nil {
		return err
	}

	fmt.Println()
	green.Printf("✓ Successfully removed %d model(s)\n\n", len(selectedIndices))
	return nil
}

func handleModelSelect() error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	if err := shared.LoadEnvFile(); err != nil {
		return fmt.Errorf("failed to load .env file: %v", err)
	}

	fmt.Println("Which model would you like to change?\n")

	modelTypeOptions := []string{
		"AI_TOOLS_MODEL    (multicoder)",
		"ASK_APP_MODEL     (ask/askc)",
	}

	choice, err := readArrowKeySelection(modelTypeOptions, "  ")
	if err != nil {
		return err
	}

	var envVar string
	switch choice {
	case 0:
		envVar = "AI_TOOLS_MODEL"
	case 1:
		envVar = "ASK_APP_MODEL"
	default:
		return fmt.Errorf("invalid choice")
	}

	cyan.Printf("\n╔════════════════════════════════════════════════════════════════╗\n")
	cyan.Printf("║  Available models for %-40s  ║\n", envVar)
	cyan.Printf("╚════════════════════════════════════════════════════════════════╝\n\n")

	modelsPath, err := modelsFilePath()
	if err != nil {
		return err
	}

	savedModels, err := loadSavedModels(modelsPath)
	if err != nil {
		return err
	}

	// Partition saved models so local entries are preserved unchanged while
	// remote entries are refreshed from the OpenRouter API.
	var localModels []ai.OpenRouterModel
	var remoteModels []ai.OpenRouterModel
	for _, m := range savedModels {
		if m.IsLocal || ai.IsLocalModel(m.ID) {
			localModels = append(localModels, m)
		} else {
			remoteModels = append(remoteModels, m)
		}
	}

	yellow.Println("→ Fetching latest data from OpenRouter for saved remote models...")

	allAPIModels, err := ai.FetchOpenRouterModels()
	if err != nil {
		return fmt.Errorf("failed to fetch models from API: %v", err)
	}

	apiModelMap := make(map[string]ai.OpenRouterModel, len(allAPIModels))
	for _, m := range allAPIModels {
		apiModelMap[m.ID] = m
	}

	var updatedRemote []ai.OpenRouterModel
	for _, saved := range remoteModels {
		if fresh, ok := apiModelMap[saved.ID]; ok {
			updatedRemote = append(updatedRemote, fresh)
		}
		// If the model no longer exists in the API we silently drop it from the
		// refresh but it will not be in mergedModels either, so it disappears
		// from the selection list.
	}

	// Merge refreshed remote models with preserved local models.
	mergedModels := append(updatedRemote, localModels...)
	sort.Slice(mergedModels, func(i, j int) bool {
		return mergedModels[i].ID < mergedModels[j].ID
	})

	if len(mergedModels) > 0 {
		if err := saveModels(modelsPath, mergedModels); err != nil {
			return err
		}
	}

	green.Printf("  ✓ Updated %d remote model(s); %d local model(s) preserved\n\n",
		len(updatedRemote), len(localModels))

	if len(mergedModels) == 0 {
		yellow.Println("No saved models found. Use 'mc model add' to add models.\n")
		return nil
	}

	// Build display options, tagging local models visually.
	modelOptions := make([]string, len(mergedModels))
	for i, m := range mergedModels {
		tag := ""
		if m.IsLocal || ai.IsLocalModel(m.ID) {
			tag = " [local]"
		}
		modelOptions[i] = m.ID + tag
	}

	modelIndex, err := readArrowKeySelection(modelOptions, "  ")
	if err != nil {
		return err
	}

	selectedModel := mergedModels[modelIndex].ID

	projectRoot, err := shared.GetProjectRoot()
	if err != nil {
		return err
	}
	envPath := filepath.Join(projectRoot, ".env")

	var lines []string
	if data, err := os.ReadFile(envPath); err == nil {
		lines = strings.Split(string(data), "\n")
	}

	updated := false
	for i, line := range lines {
		if strings.HasPrefix(line, envVar+"=") {
			lines[i] = fmt.Sprintf("%s=%s", envVar, selectedModel)
			updated = true
			break
		}
	}

	if !updated {
		lines = append(lines, fmt.Sprintf("%s=%s", envVar, selectedModel))
	}

	if err := os.WriteFile(envPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return err
	}

	fmt.Println()
	green.Printf("✓ Updated %s to:\n", envVar)
	cyan.Printf("  %s\n", selectedModel)
	yellow.Printf("\n→ Location: %s\n\n", envPath)

	return nil
}

func readArrowKeySelection(options []string, prefix string) (int, error) {
	execCmd := func(cmd string, args ...string) error {
		c := exec.Command(cmd, args...)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	}

	if err := execCmd("stty", "-echo", "-icanon"); err != nil {
		return 0, err
	}
	defer execCmd("stty", "echo", "icanon")

	currentSelection := 0
	viewOffset := 0
	green := color.New(color.FgGreen)
	white := color.New(color.FgWhite)
	yellow := color.New(color.FgYellow)

	termHeight := getTerminalHeight()
	visibleLines := termHeight - 8

	printOptions := func() {
		fmt.Print("\033[H\033[2J")

		cyan := color.New(color.FgCyan, color.Bold)
		cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
		cyan.Println("║                       MODEL SELECTION                          ║")
		cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

		endIdx := viewOffset + visibleLines
		if endIdx > len(options) {
			endIdx = len(options)
		}

		for i := viewOffset; i < endIdx; i++ {
			option := options[i]
			if i == currentSelection {
				green.Printf("%s→ %s\n", prefix, option)
			} else if strings.Contains(option, "[local]") {
				yellow.Printf("%s  %s\n", prefix, option)
			} else {
				white.Printf("%s  %s\n", prefix, option)
			}
		}
		fmt.Println()
		if len(options) > visibleLines {
			yellow.Printf("  Viewing: %d-%d of %d\n", viewOffset+1, endIdx, len(options))
		}
	}

	printOptions()

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return 0, err
		}

		if n == 1 && (buf[0] == 13 || buf[0] == 10) {
			fmt.Print("\033[H\033[2J")
			return currentSelection, nil
		}

		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 65: // Up
				if currentSelection > 0 {
					currentSelection--
					if currentSelection < viewOffset {
						viewOffset = currentSelection
					}
					printOptions()
				}
			case 66: // Down
				if currentSelection < len(options)-1 {
					currentSelection++
					if currentSelection >= viewOffset+visibleLines {
						viewOffset = currentSelection - visibleLines + 1
					}
					printOptions()
				}
			}
		}
	}
}

func getTerminalHeight() int {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return 30
	}
	var rows, cols int
	fmt.Sscanf(string(out), "%d %d", &rows, &cols)
	if rows > 0 {
		return rows
	}
	return 30
}
