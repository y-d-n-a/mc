
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"mc/pkg/ai"
	"mc/pkg/shared"

	"github.com/fatih/color"
)

func main() {
	if err := shared.LoadEnvFile(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load .env file: %v\n", err)
	}

	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	model := os.Getenv("ASK_APP_MODEL")
	if model == "" {
		fmt.Fprintf(os.Stderr, "Error: ASK_APP_MODEL is not set in .env\n")
		os.Exit(1)
	}

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                    INTERACTIVE CHAT MODE                       ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝")
	yellow.Println("\nType 'exit' or 'quit' to end the conversation\n")

	mi, err := ai.NewModelInterface("", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing AI: %v\n", err)
		os.Exit(1)
	}

	var messages []ai.Message
	messages = append(messages, ai.Message{
		Role:    "system",
		Content: "You are a helpful assistant.",
	})

	reader := bufio.NewReader(os.Stdin)

	for {
		green.Print("You: ")
		userInput, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			continue
		}

		userInput = strings.TrimSpace(userInput)

		if userInput == "exit" || userInput == "quit" {
			cyan.Println("\nGoodbye!")
			break
		}

		if userInput == "" {
			continue
		}

		messages = append(messages, ai.Message{
			Role:    "user",
			Content: userInput,
		})

		response, err := mi.SendToAI("", model, 0, 0.7, "", messages)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		messages = append(messages, ai.Message{
			Role:    "assistant",
			Content: response,
		})

		cyan.Print("\nAssistant: ")
		fmt.Println(response)
		fmt.Println()
	}
}
