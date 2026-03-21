
package main

import (
        "fmt"
        "os"
        "strconv"

        "mc/pkg/multicoder"
)

func main() {
        if len(os.Args) < 2 {
                printUsage()
                os.Exit(1)
        }

        command := os.Args[1]

        switch command {
        case "get":
                if len(os.Args) < 4 {
                        fmt.Println("Usage: mc get <llm_count> <file|pattern> [file|pattern ...] [-r] [-- instructions]")
                        os.Exit(1)
                }
                llmCount, err := strconv.Atoi(os.Args[2])
                if err != nil {
                        fmt.Printf("Invalid llm_count: %v\n", err)
                        os.Exit(1)
                }

                recursive := false
                var targets []string
                userInstructions := ""
                pastDash := false

                for i := 3; i < len(os.Args); i++ {
                        arg := os.Args[i]
                        if pastDash {
                                if userInstructions == "" {
                                        userInstructions = arg
                                } else {
                                        userInstructions += " " + arg
                                }
                                continue
                        }
                        if arg == "--" {
                                pastDash = true
                                continue
                        }
                        if arg == "-r" {
                                recursive = true
                                continue
                        }
                        targets = append(targets, arg)
                }

                if len(targets) == 0 {
                        fmt.Println("Error: no file targets specified")
                        os.Exit(1)
                }

                if err := multicoder.HandleGet(llmCount, targets, recursive, userInstructions); err != nil {
                        fmt.Printf("Error: %v\n", err)
                        os.Exit(1)
                }

        case "write":
                if len(os.Args) < 3 {
                        fmt.Println("Usage: mc write <response_index|list>")
                        os.Exit(1)
                }
                arg := os.Args[2]
                if arg == "list" {
                        if err := multicoder.ListResponses(); err != nil {
                                fmt.Printf("Error: %v\n", err)
                                os.Exit(1)
                        }
                } else {
                        responseIndex, err := strconv.Atoi(arg)
                        if err != nil {
                                fmt.Printf("Invalid response index: %v\n", err)
                                os.Exit(1)
                        }
                        if err := multicoder.HandleWrite(responseIndex); err != nil {
                                fmt.Printf("Error: %v\n", err)
                                os.Exit(1)
                        }
                }

        case "open":
                if len(os.Args) == 3 {
                        responseIndex, err := strconv.Atoi(os.Args[2])
                        if err != nil {
                                fmt.Printf("Invalid response index: %v\n", err)
                                os.Exit(1)
                        }
                        if err := multicoder.OpenResponse(responseIndex); err != nil {
                                fmt.Printf("Error: %v\n", err)
                                os.Exit(1)
                        }
                } else {
                        if err := multicoder.OpenResponses(); err != nil {
                                fmt.Printf("Error: %v\n", err)
                                os.Exit(1)
                        }
                }

        case "checkpoint":
                if err := multicoder.SetCheckpoint(); err != nil {
                        fmt.Printf("Error: %v\n", err)
                        os.Exit(1)
                }

        case "rollback":
                if len(os.Args) == 3 && os.Args[2] == "checkpoint" {
                        if err := multicoder.RollbackToCheckpoint(); err != nil {
                                fmt.Printf("Error: %v\n", err)
                                os.Exit(1)
                        }
                } else if len(os.Args) == 3 {
                        n, err := strconv.Atoi(os.Args[2])
                        if err != nil {
                                fmt.Printf("Invalid version number: %v\n", err)
                                os.Exit(1)
                        }
                        if err := multicoder.HandleRollback(&n); err != nil {
                                fmt.Printf("Error: %v\n", err)
                                os.Exit(1)
                        }
                } else {
                        if err := multicoder.HandleRollback(nil); err != nil {
                                fmt.Printf("Error: %v\n", err)
                                os.Exit(1)
                        }
                }

        case "clear":
                confirm := false
                if len(os.Args) > 2 && os.Args[2] == "-y" {
                        confirm = true
                }
                if err := multicoder.ClearWorkspace(confirm); err != nil {
                        fmt.Printf("Error: %v\n", err)
                        os.Exit(1)
                }

        case "undo":
                if err := multicoder.UndoLastWrite(); err != nil {
                        fmt.Printf("Error: %v\n", err)
                        os.Exit(1)
                }

        case "ignore":
                if len(os.Args) < 3 {
                        fmt.Println("Usage: mc ignore <pattern>")
                        os.Exit(1)
                }
                if err := multicoder.Ignore(os.Args[2]); err != nil {
                        fmt.Printf("Error: %v\n", err)
                        os.Exit(1)
                }

        case "rmignore":
                if len(os.Args) < 3 {
                        fmt.Println("Usage: mc rmignore <pattern>")
                        os.Exit(1)
                }
                if err := multicoder.Unignore(os.Args[2]); err != nil {
                        fmt.Printf("Error: %v\n", err)
                        os.Exit(1)
                }

        case "lsignores":
                if err := multicoder.Lsignores(); err != nil {
                        fmt.Printf("Error: %v\n", err)
                        os.Exit(1)
                }

        case "model":
                subcommand := ""
                if len(os.Args) > 2 {
                        subcommand = os.Args[2]
                }
                if err := multicoder.HandleModel(subcommand, ""); err != nil {
                        fmt.Printf("Error: %v\n", err)
                        os.Exit(1)
                }

        case "repeat":
                repeatCount := 1
                if len(os.Args) > 2 {
                        var err error
                        repeatCount, err = strconv.Atoi(os.Args[2])
                        if err != nil {
                                fmt.Printf("Invalid repeat count: %v\n", err)
                                os.Exit(1)
                        }
                }
                if err := multicoder.HandleRepeat(repeatCount); err != nil {
                        fmt.Printf("Error: %v\n", err)
                        os.Exit(1)
                }

        case "prompt":
                if len(os.Args) < 3 {
                        fmt.Println("Usage: mc prompt <add|delete|update|switch|list> [name]")
                        os.Exit(1)
                }
                subcommand := os.Args[2]
                name := ""
                if len(os.Args) > 3 {
                        name = os.Args[3]
                }

                switch subcommand {
                case "add":
                        if name == "" {
                                fmt.Println("Usage: mc prompt add <name>")
                                os.Exit(1)
                        }
                        if err := multicoder.HandlePromptAdd(name); err != nil {
                                fmt.Printf("Error: %v\n", err)
                                os.Exit(1)
                        }
                case "delete":
                        if name == "" {
                                fmt.Println("Usage: mc prompt delete <name>")
                                os.Exit(1)
                        }
                        if err := multicoder.HandlePromptDelete(name); err != nil {
                                fmt.Printf("Error: %v\n", err)
                                os.Exit(1)
                        }
                case "update":
                        if name == "" {
                                fmt.Println("Usage: mc prompt update <name>")
                                os.Exit(1)
                        }
                        if err := multicoder.HandlePromptUpdate(name); err != nil {
                                fmt.Printf("Error: %v\n", err)
                                os.Exit(1)
                        }
                case "switch":
                        if err := multicoder.HandlePromptSwitch(name); err != nil {
                                fmt.Printf("Error: %v\n", err)
                                os.Exit(1)
                        }
                case "list":
                        if err := multicoder.HandlePromptList(); err != nil {
                                fmt.Printf("Error: %v\n", err)
                                os.Exit(1)
                        }
                default:
                        fmt.Printf("Unknown prompt subcommand: %s\n", subcommand)
                        fmt.Println("Available: add, delete, update, switch, list")
                        os.Exit(1)
                }

        case "cost":
                if len(os.Args) > 2 && os.Args[2] == "clear" {
                        if err := multicoder.ClearProjectCost(); err != nil {
                                fmt.Printf("Error: %v\n", err)
                                os.Exit(1)
                        }
                } else {
                        if err := multicoder.ShowProjectCost(); err != nil {
                                fmt.Printf("Error: %v\n", err)
                                os.Exit(1)
                        }
                }

        case "watch":
                pattern := ""
                recursive := false

                for i := 2; i < len(os.Args); i++ {
                        arg := os.Args[i]
                        if arg == "-r" {
                                recursive = true
                                continue
                        }
                        if pattern == "" {
                                pattern = arg
                        }
                }

                if pattern == "" {
                        fmt.Println("Usage: mc watch [-r] <pattern>")
                        fmt.Println("  Example: mc watch \"*.go\"")
                        fmt.Println("  Example: mc watch -r \"*.go\"")
                        os.Exit(1)
                }

                if err := multicoder.HandleWatch(pattern, recursive); err != nil {
                        fmt.Printf("Error: %v\n", err)
                        os.Exit(1)
                }

        default:
                fmt.Printf("Unknown command: %s\n", command)
                printUsage()
                os.Exit(1)
        }
}

func printUsage() {
        fmt.Println("Usage: mc <command> [options]")
        fmt.Println("\nCommands:")
        fmt.Println("  get <count> <file|glob> [file|glob ...] [-r] [-- instructions]")
        fmt.Println("                                              - Get files and send to LLMs")
        fmt.Println("  write <index|list>                          - Write response to disk")
        fmt.Println("  open [index]                                - View response(s)")
        fmt.Println("  checkpoint                                  - Set checkpoint at current version")
        fmt.Println("  rollback [n|checkpoint]                     - Rollback to version")
        fmt.Println("  clear [-y]                                  - Clear workspace")
        fmt.Println("  undo                                        - Undo last write")
        fmt.Println("  ignore <pattern>                            - Add ignore pattern")
        fmt.Println("  rmignore <pattern>                          - Remove ignore pattern")
        fmt.Println("  lsignores                                   - List ignore patterns")
        fmt.Println("  model [add|remove]                          - Manage models")
        fmt.Println("  repeat [count]                              - Repeat last call")
        fmt.Println("  prompt <add|delete|update|switch|list>      - Manage system prompts")
        fmt.Println("  cost [clear]                                - Show or clear project costs")
        fmt.Println("  watch [-r] <pattern>                        - Watch files for AI prompt tags")
}
