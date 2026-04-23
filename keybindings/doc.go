// Package keybindings provides keyboard shortcut configuration management for Claude Code.
// 
// This package mirrors the functionality of the TypeScript keybindings system,
// providing the ability to load, parse, validate, and manage user keybinding configurations.
//
// Key Features:
// - Loading and parsing of ~/.claude/keybindings.json configuration files
// - Validation of keybinding syntax and conflict detection
// - Template generation for new keybinding configurations  
// - Support for chord bindings (multi-keystroke sequences)
// - Integration with the /keybindings command to open configuration in editor
//
// File Structure:
// The keybindings.json file follows this structure:
//   {
//     "$schema": "https://www.schemastore.org/claude-code-keybindings.json",
//     "$docs": "https://code.claude.com/docs/en/keybindings",
//     "bindings": [
//       {
//         "context": "Chat",
//         "bindings": {
//           "ctrl+e": "chat:externalEditor",
//           "ctrl+s": null
//         }
//       }
//     ]
//   }
//
// Usage:
//   // Load user keybindings
//   result, err := keybindings.LoadKeybindings()
//   if err != nil {
//     log.Fatal(err)
//   }
//   
//   // Handle the /keybindings command
//   response, err := keybindings.ExecuteKeybindingsCommand()
//   if err != nil {
//     log.Fatal(err)
//   }
package keybindings