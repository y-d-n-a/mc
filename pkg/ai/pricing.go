
package ai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"mc/pkg/shared"
)

type Pricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
	Request    string `json:"request"`
	Image      string `json:"image"`
}

type Architecture struct {
	Modality         string   `json:"modality"`
	InputModalities  []string `json:"input_modalities"`
	OutputModalities []string `json:"output_modalities"`
	Tokenizer        string   `json:"tokenizer"`
	InstructType     string   `json:"instruct_type"`
}

type TopProvider struct {
	IsModerated         bool `json:"is_moderated"`
	ContextLength       int  `json:"context_length"`
	MaxCompletionTokens int  `json:"max_completion_tokens"`
}

// OpenRouterModel represents a single entry in models.json.
//
// IsLocal is true for entries that route to LOCAL_URL instead of OpenRouter.
// It is set automatically when a model ID starts with "ollama/".
// Remote entries fetched from the OpenRouter API will never have IsLocal set.
type OpenRouterModel struct {
	ID                  string       `json:"id"`
	CanonicalSlug       string       `json:"canonical_slug,omitempty"`
	Name                string       `json:"name"`
	Created             int64        `json:"created,omitempty"`
	Pricing             Pricing      `json:"pricing"`
	ContextLength       int          `json:"context_length,omitempty"`
	Architecture        Architecture `json:"architecture,omitempty"`
	TopProvider         TopProvider  `json:"top_provider,omitempty"`
	PerRequestLimits    interface{}  `json:"per_request_limits,omitempty"`
	SupportedParameters []string     `json:"supported_parameters,omitempty"`
	DefaultParameters   interface{}  `json:"default_parameters,omitempty"`
	Description         string       `json:"description,omitempty"`
	ExpirationDate      interface{}  `json:"expiration_date,omitempty"`
	IsLocal             bool         `json:"is_local,omitempty"`
}

type OpenRouterResponse struct {
	Data []OpenRouterModel `json:"data"`
}

// modelsFileName is the canonical name of the local models catalogue.
const modelsFileName = "models.json"

func getModelsPath() (string, error) {
	projectRoot, err := shared.GetProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectRoot, modelsFileName), nil
}

// FetchOpenRouterModels retrieves the full model list from the OpenRouter API.
// It honours the OPENROUTER_URL env var (defaults to https://openrouter.ai/api/v1).
func FetchOpenRouterModels() ([]OpenRouterModel, error) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY not set")
	}

	baseURL := os.Getenv("OPENROUTER_URL")
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}
	baseURL = trimTrailingSlash(baseURL)
	url := baseURL + "/models"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openrouter API failed with status %d: %s", resp.StatusCode, string(body))
	}

	var openRouterResp OpenRouterResponse
	if err = json.NewDecoder(resp.Body).Decode(&openRouterResp); err != nil {
		return nil, err
	}

	return openRouterResp.Data, nil
}

func SaveModelsToJSON(models []OpenRouterModel) error {
	modelsPath, err := getModelsPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(models, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(modelsPath, data, 0644)
}

func LoadModelsFromJSON() ([]OpenRouterModel, error) {
	modelsPath, err := getModelsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(modelsPath)
	if err != nil {
		return nil, err
	}

	var models []OpenRouterModel
	if err = json.Unmarshal(data, &models); err != nil {
		return nil, err
	}

	// Back-fill IsLocal for any entry whose ID starts with "ollama/" in case the
	// field was omitted when the entry was written manually.
	for i := range models {
		if strings.HasPrefix(models[i].ID, "ollama/") {
			models[i].IsLocal = true
		}
	}

	return models, nil
}

// GetOpenRouterModels returns saved models, falling back to a live API fetch
// if the local file is missing or empty.
func GetOpenRouterModels() ([]OpenRouterModel, error) {
	models, err := LoadModelsFromJSON()
	if err == nil && len(models) > 0 {
		return models, nil
	}

	models, err = FetchOpenRouterModels()
	if err != nil {
		return nil, err
	}

	if err := SaveModelsToJSON(models); err != nil {
		return models, nil
	}

	return models, nil
}

// GetModelPricing looks up prompt and completion prices for a model ID.
// Local (ollama/) models always return 0, 0 with no error.
func GetModelPricing(modelID string) (promptPrice float64, completionPrice float64, err error) {
	// Short-circuit for local models — they have no API cost.
	if strings.HasPrefix(modelID, "ollama/") {
		return 0, 0, nil
	}

	models, err := LoadModelsFromJSON()
	if err != nil {
		return 0, 0, err
	}

	for _, model := range models {
		if model.ID == modelID {
			var p, c float64
			fmt.Sscanf(model.Pricing.Prompt, "%f", &p)
			fmt.Sscanf(model.Pricing.Completion, "%f", &c)
			return p, c, nil
		}
	}

	return 0, 0, fmt.Errorf("model %s not found in models.json", modelID)
}

// NewLocalModel constructs a minimal OpenRouterModel entry for a local model.
// The id must already include the "ollama/" prefix.
func NewLocalModel(id string) OpenRouterModel {
	return OpenRouterModel{
		ID:      id,
		Name:    id,
		IsLocal: true,
		Pricing: Pricing{
			Prompt:     "0",
			Completion: "0",
			Request:    "",
			Image:      "",
		},
	}
}

func trimTrailingSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
