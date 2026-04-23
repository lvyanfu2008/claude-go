// Package main provides a demonstration of the keybindings functionality
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"goc/keybindings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: keybindings-demo <command>")
		fmt.Println("Commands:")
		fmt.Println("  load     - Load and display current keybindings")
		fmt.Println("  validate - Validate keybindings configuration")
		fmt.Println("  create   - Create a template keybindings.json file")
		fmt.Println("  command  - Execute the /keybindings command")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "load":
		loadKeybindings()
	case "validate":
		validateKeybindings()
	case "create":
		createTemplate()
	case "command":
		executeCommand()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func loadKeybindings() {
	fmt.Println("Loading keybindings...")

	result, err := keybindings.LoadKeybindings()
	if err != nil {
		log.Fatalf("Failed to load keybindings: %v", err)
	}

	fmt.Printf("Loaded %d keybindings\n", len(result.Bindings))
	if len(result.Warnings) > 0 {
		fmt.Printf("Found %d warnings:\n", len(result.Warnings))
		for _, warning := range result.Warnings {
			fmt.Printf("  [%s] %s: %s\n", warning.Severity, warning.Type, warning.Message)
		}
	}

	// Display some sample bindings
	fmt.Println("\nSample bindings:")
	contextCounts := make(map[string]int)
	for _, binding := range result.Bindings {
		contextCounts[string(binding.Context)]++
	}

	for context, count := range contextCounts {
		fmt.Printf("  %s: %d bindings\n", context, count)
	}
}

func validateKeybindings() {
	fmt.Println("Validating keybindings configuration...")

	result, err := keybindings.LoadKeybindings()
	if err != nil {
		log.Fatalf("Failed to load keybindings: %v", err)
	}

	if len(result.Warnings) == 0 {
		fmt.Println("✓ No validation issues found!")
		return
	}

	fmt.Printf("Found %d validation issues:\n", len(result.Warnings))
	for _, warning := range result.Warnings {
		icon := "⚠️"
		if warning.Severity == "error" {
			icon = "❌"
		}
		fmt.Printf("%s [%s] %s\n", icon, warning.Severity, warning.Message)
		if warning.Context != "" {
			fmt.Printf("    Context: %s\n", warning.Context)
		}
		if warning.Key != "" {
			fmt.Printf("    Key: %s\n", warning.Key)
		}
		if warning.Suggestion != "" {
			fmt.Printf("    Suggestion: %s\n", warning.Suggestion)
		}
		fmt.Println()
	}
}

func createTemplate() {
	path, err := keybindings.GetKeybindingsPath()
	if err != nil {
		log.Fatalf("Failed to get keybindings path: %v", err)
	}

	fmt.Printf("Creating keybindings template at %s...\n", path)

	if err := keybindings.SaveKeybindingsTemplate(path); err != nil {
		log.Fatalf("Failed to create template: %v", err)
	}

	fmt.Println("✓ Template created successfully!")
	fmt.Printf("Edit the file to customize your keybindings: %s\n", path)
}

func executeCommand() {
	fmt.Println("Executing /keybindings command...")

	result, err := keybindings.ExecuteKeybindingsCommand()
	if err != nil {
		log.Fatalf("Failed to execute command: %v", err)
	}

	// Pretty print the result
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal result: %v", err)
	}

	fmt.Println("Command result:")
	fmt.Println(string(resultJSON))
}