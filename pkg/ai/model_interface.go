
package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"mc/pkg/shared"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
}

type ChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type ModelInterface struct {
	CostTracker   *CostTracker
	OpenRouterKey string
	MaxTokens     int
	Timeout       time.Duration
}

func getEnvInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 {
		return defaultVal
	}
	return n
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 {
		return defaultVal
	}
	return time.Duration(n) * time.Second
}

// IsLocalModel returns true when the model ID starts with "ollama/", indicating
// the request must be sent to LOCAL_URL instead of OpenRouter.
func IsLocalModel(model string) bool {
	return strings.HasPrefix(model, "ollama/")
}

// StripLocalPrefix removes the "ollama/" provider prefix before sending the
// model name to the local endpoint, which expects just the bare model name.
func StripLocalPrefix(model string) string {
	return strings.TrimPrefix(model, "ollama/")
}

// openRouterBaseURL returns the base URL for OpenRouter, stripping any trailing
// slash. Falls back to the canonical base if the env var is not set.
func openRouterBaseURL() string {
	base := os.Getenv("OPENROUTER_URL")
	if base == "" {
		base = "https://openrouter.ai/api/v1"
	}
	for len(base) > 0 && base[len(base)-1] == '/' {
		base = base[:len(base)-1]
	}
	return base
}

func NewModelInterface(clusterConfigPath, countersPath string) (*ModelInterface, error) {
	if err := shared.LoadEnvFile(); err != nil {
		return nil, fmt.Errorf("failed to load .env file: %v", err)
	}

	maxTokens := getEnvInt("MAX_TOKENS", 4096)
	timeout := getEnvDuration("TIMEOUT", 120*time.Second)

	mi := &ModelInterface{
		CostTracker:   NewCostTracker(),
		OpenRouterKey: os.Getenv("OPENROUTER_API_KEY"),
		MaxTokens:     maxTokens,
		Timeout:       timeout,
	}

	return mi, nil
}

// SendToAI dispatches the request to either the local cluster or OpenRouter
// based on whether the model ID starts with "ollama/".
func (mi *ModelInterface) SendToAI(prompt, model string, maxTokens int, temperature float64, systemPrompt string, messages []Message) (string, error) {
	effectiveMaxTokens := mi.MaxTokens
	if maxTokens > 0 && maxTokens != mi.MaxTokens {
		effectiveMaxTokens = maxTokens
	}

	if IsLocalModel(model) {
		return mi.invokeLocal(prompt, model, effectiveMaxTokens, temperature, systemPrompt, messages)
	}
	return mi.invokeOpenRouter(prompt, model, effectiveMaxTokens, temperature, systemPrompt, messages)
}

func (mi *ModelInterface) invokeOpenRouter(prompt, model string, maxTokens int, temperature float64, systemPrompt string, messages []Message) (string, error) {
	if mi.OpenRouterKey == "" {
		return "", fmt.Errorf("OPENROUTER_API_KEY not set in .env")
	}

	url := openRouterBaseURL() + "/chat/completions"

	var reqMessages []Message
	if messages != nil {
		reqMessages = messages
	} else {
		reqMessages = []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		}
	}

	chatReq := ChatRequest{
		Model:       model,
		Messages:    reqMessages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	jsonData, err := json.Marshal(chatReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request JSON: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+mi.OpenRouterKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: mi.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to OpenRouter: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openrouter API failed with status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err = json.NewDecoder(bytes.NewReader(body)).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to parse OpenRouter response JSON: %v. Response body: %s", err, string(body))
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from API")
	}

	inputTokens := chatResp.Usage.PromptTokens
	outputTokens := chatResp.Usage.CompletionTokens
	mi.CostTracker.AddRequestMetrics(model, inputTokens, outputTokens)

	return chatResp.Choices[0].Message.Content, nil
}

func (mi *ModelInterface) invokeLocal(prompt, model string, maxTokens int, temperature float64, systemPrompt string, messages []Message) (string, error) {
	url := os.Getenv("LOCAL_URL")
	if url == "" {
		return "", fmt.Errorf("LOCAL_URL not set in .env — required for local model routing (model: %s)", model)
	}

	// Strip the "ollama/" prefix; the local endpoint expects just the model name.
	localModelName := StripLocalPrefix(model)

	var reqMessages []Message
	if messages != nil {
		reqMessages = messages
	} else {
		reqMessages = []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		}
	}

	chatReq := ChatRequest{
		Model:       localModelName,
		Messages:    reqMessages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	jsonData, err := json.Marshal(chatReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request JSON: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if mi.OpenRouterKey != "" {
		req.Header.Set("Authorization", "Bearer "+mi.OpenRouterKey)
	}

	client := &http.Client{Timeout: mi.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("local cluster request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("local cluster API failed with status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err = json.NewDecoder(bytes.NewReader(body)).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to parse local response JSON: %v. Response body: %s", err, string(body))
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from local cluster")
	}

	inputTokens := chatResp.Usage.PromptTokens
	outputTokens := chatResp.Usage.CompletionTokens
	mi.CostTracker.AddRequestMetrics(model, inputTokens, outputTokens)

	return chatResp.Choices[0].Message.Content, nil
}
