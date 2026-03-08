
package ai

import (
	"fmt"

	"github.com/fatih/color"
)

type CostData struct {
	Model      string  `json:"model"`
	InputCost  float64 `json:"input_cost"`
	OutputCost float64 `json:"output_cost"`
	TotalCost  float64 `json:"total_cost"`
}

type CostTracker struct {
	CostData []CostData
}

func NewCostTracker() *CostTracker {
	return &CostTracker{
		CostData: make([]CostData, 0),
	}
}

func (ct *CostTracker) ShowCostData() {
	totalOfTotals := 0.0

	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                        COST BREAKDOWN                          ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝")

	fmt.Printf("\n%-10s %-20s %-15s %-15s\n",
		cyan.Sprint("Index"),
		cyan.Sprint("Input Cost"),
		cyan.Sprint("Output Cost"),
		cyan.Sprint("Total Cost"))
	fmt.Println(color.HiBlackString("─────────────────────────────────────────────────────────────────"))

	for index, cost := range ct.CostData {
		indexStr := yellow.Sprintf("%-10d", index)
		inputStr := green.Sprintf("$%-19.6f", cost.InputCost)
		outputStr := green.Sprintf("$%-14.6f", cost.OutputCost)
		totalStr := fmt.Sprintf("$%-14.6f", cost.TotalCost)

		fmt.Printf("%s %s %s %s\n", indexStr, inputStr, outputStr, totalStr)
		totalOfTotals += cost.TotalCost
	}

	fmt.Println(color.HiBlackString("─────────────────────────────────────────────────────────────────"))
	
	var cumulativeColor *color.Color
	if totalOfTotals > 1.0 {
		cumulativeColor = red
	} else if totalOfTotals > 0.1 {
		cumulativeColor = yellow
	} else {
		cumulativeColor = green
	}
	
	fmt.Printf("\n%s %s\n\n", 
		cyan.Sprint("Cumulative Cost:"), 
		cumulativeColor.Sprintf("$%.6f", totalOfTotals))
}

func (ct *CostTracker) AddActualCost(model string, totalCost float64, inputTokens, outputTokens int) {
	// Calculate proportional input/output costs based on token counts
	// This is an approximation since OpenRouter gives us total_cost
	var inputCost, outputCost float64
	
	if inputTokens > 0 && outputTokens > 0 {
		// Get the pricing to calculate ratio
		promptPrice, completionPrice, err := GetModelPricing(model)
		if err == nil && (promptPrice > 0 || completionPrice > 0) {
			// Calculate expected costs
			expectedInputCost := promptPrice * float64(inputTokens)
			expectedOutputCost := completionPrice * float64(outputTokens)
			expectedTotal := expectedInputCost + expectedOutputCost
			
			if expectedTotal > 0 {
				// Distribute actual cost proportionally
				inputCost = totalCost * (expectedInputCost / expectedTotal)
				outputCost = totalCost * (expectedOutputCost / expectedTotal)
			} else {
				// Fallback: assume 1:3 ratio (typical for most models)
				inputCost = totalCost * 0.25
				outputCost = totalCost * 0.75
			}
		} else {
			// Fallback: assume 1:3 ratio
			inputCost = totalCost * 0.25
			outputCost = totalCost * 0.75
		}
	} else {
		// If we don't have token counts, split evenly
		inputCost = totalCost * 0.5
		outputCost = totalCost * 0.5
	}

	costData := CostData{
		Model:      model,
		InputCost:  inputCost,
		OutputCost: outputCost,
		TotalCost:  totalCost,
	}
	
	ct.CostData = append(ct.CostData, costData)
}

func (ct *CostTracker) AddRequestMetrics(model string, inputTokens, outputTokens int) {
	costData := ct.CalculateRequestCosts(model, inputTokens, outputTokens)
	ct.CostData = append(ct.CostData, costData)
}

func (ct *CostTracker) CalculateRequestCosts(model string, inputTokens, outputTokens int) CostData {
	var inputCostPerToken, outputCostPerToken float64

	promptPrice, completionPrice, err := GetModelPricing(model)
	if err == nil && (promptPrice > 0 || completionPrice > 0) {
		inputCostPerToken = promptPrice
		outputCostPerToken = completionPrice
	} else {
		modelCosts := map[string]struct{ Input, Output float64 }{
			"openai/gpt-4o":                {0.000005, 0.000015},
			"openai/gpt-4-turbo":           {0.000005, 0.000015},
			"openai/gpt-3.5-turbo":         {0.0000005, 0.0000015},
			"openai/gpt-4o-mini":           {0.00000015, 0.0000006},
			"anthropic/claude-3-opus":      {0.000015, 0.000075},
			"anthropic/claude-3-sonnet":    {0.000003, 0.000015},
			"anthropic/claude-3-haiku":     {0.00000025, 0.00000125},
			"anthropic/claude-3-5-sonnet":  {0.000003, 0.000015},
			"anthropic/claude-3-7-sonnet":  {0.000003, 0.000015},
			"anthropic/claude-sonnet-4":    {0.000003, 0.000015},
			"default":                      {0.000001, 0.000002},
		}

		costs, ok := modelCosts[model]
		if !ok {
			costs = modelCosts["default"]
		}
		inputCostPerToken = costs.Input
		outputCostPerToken = costs.Output
	}

	inputCost := inputCostPerToken * float64(inputTokens)
	outputCost := outputCostPerToken * float64(outputTokens)
	totalCost := inputCost + outputCost

	return CostData{
		Model:      model,
		InputCost:  inputCost,
		OutputCost: outputCost,
		TotalCost:  totalCost,
	}
}
