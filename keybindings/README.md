# Keybindings Package

Go implementation of Claude Code's keyboard shortcut configuration system, providing feature parity with the TypeScript keybindings functionality.

## Features

✅ **Complete functionality implemented:**

- **Configuration File Management**
  - Load and parse `~/.claude/keybindings.json` 
  - Generate keybindings template with default bindings
  - Validate JSON structure and syntax

- **Keystroke Parsing** 
  - Support for single keystrokes: `ctrl+s`, `alt+enter`
  - Support for chord bindings: `ctrl+k ctrl+s`
  - Modifier key aliases: `cmd`/`meta`, `opt`/`alt`, `ctrl`/`control`
  - Special key aliases: `esc`/`escape`, `return`/`enter`

- **Validation System**
  - Reserved shortcut detection (non-rebindable keys)
  - Conflict detection with terminal/OS shortcuts
  - Duplicate binding warnings
  - Context and action validation

- **Command Integration**
  - `/keybindings` command to open configuration in editor
  - Auto-detection of available editors (VS Code, vim, nano, etc.)
  - Platform-specific editor defaults (macOS, Linux, Windows)

## Usage

### Loading Keybindings
```go
result, err := keybindings.LoadKeybindings()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Loaded %d keybindings with %d warnings\n", 
    len(result.Bindings), len(result.Warnings))
```

### Creating Template
```go
path, _ := keybindings.GetKeybindingsPath()
err := keybindings.SaveKeybindingsTemplate(path)
```

### Executing /keybindings Command
```go
result, err := keybindings.ExecuteKeybindingsCommand()
// Opens keybindings.json in user's preferred editor
```

## Demo Tool

Build and run the demo to test functionality:

```bash
cd claude-go
go build ./cmd/keybindings-demo

# Load and display current keybindings
./keybindings-demo load

# Validate configuration
./keybindings-demo validate  

# Create template file
./keybindings-demo create

# Execute /keybindings command
./keybindings-demo command
```

## Configuration Format

The `~/.claude/keybindings.json` file follows this structure:

```json
{
  "$schema": "https://www.schemastore.org/claude-code-keybindings.json",
  "$docs": "https://code.claude.com/docs/en/keybindings", 
  "bindings": [
    {
      "context": "Chat",
      "bindings": {
        "ctrl+e": "chat:externalEditor",
        "ctrl+s": null,
        "ctrl+k ctrl+s": "app:save"
      }
    }
  ]
}
```

## Integration

The keybindings system integrates with Claude Code's command system:

1. **Command Definition**: The `keybindings` command is defined in `commands/handwritten/z_builtin_table_gen.go`
2. **Command Handler**: Local command routing in `commands/handlers/`
3. **Core Logic**: All keybinding functionality in `keybindings/` package

## Testing

Run tests to verify functionality:
```bash
go test ./keybindings/...
```

## Architecture 

- `types.go` - Core data structures and types
- `defaults.go` - Default keybinding definitions
- `reserved.go` - Reserved shortcut definitions  
- `config.go` - Configuration file I/O
- `parser.go` - Keystroke parsing logic
- `validate.go` - Validation and warning system
- `command.go` - Command execution logic

## Status

This implementation provides complete feature parity with the TypeScript keybindings system and is ready for production use.