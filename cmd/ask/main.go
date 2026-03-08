
package main

import (
	"fmt"
	"os"
	"strings"

	"mc/pkg/ai"
	"mc/pkg/shared"
)

func main() {
	if err := shared.LoadEnvFile(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load .env file: %v\n", err)
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage: ask <prompt>")
		os.Exit(1)
	}

	prompt := strings.Join(os.Args[1:], " ")

	model := os.Getenv("ASK_APP_MODEL")
	if model == "" {
		fmt.Fprintf(os.Stderr, "Error: ASK_APP_MODEL is not set in .env\n")
		os.Exit(1)
	}

	mi, err := ai.NewModelInterface("", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing AI: %v\n", err)
		os.Exit(1)
	}

	response, err := mi.SendToAI(prompt, model, 0, 0.7, "You are a helpful assistant.", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(response)
}
