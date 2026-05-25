package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	plugins "goc/plugincore"
	"github.com/spf13/cobra"
)

// ---- plugin / plugins ----

var pluginCmd = &cobra.Command{
	Use:     "plugin",
	Aliases: []string{"plugins"},
	Short:   "Manage Claude Code plugins",
	Long:    "Validate, install, uninstall, enable, disable, and list plugins.",
}

// --- plugin validate ---

var pluginValidateCmd = &cobra.Command{
	Use:   "validate <path>",
	Short: "Validate a plugin manifest",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("path not found: %s", path)
		}
		fmt.Printf("Plugin validation for %q not yet implemented in Go.\n", path)
		return nil
	},
}

// --- plugin list ---

var pluginListJSON bool
var pluginListAvailable bool

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	RunE: func(cmd *cobra.Command, args []string) error {
		store := plugins.NewDiskStore()
		cacheDir, err := pluginCacheDir()
		if err != nil {
			return err
		}
		installed, err := store.ListInstalled(cmd.Context(), cacheDir)
		if err != nil {
			return err
		}
		if len(installed) == 0 {
			fmt.Println("No plugins installed.")
			return nil
		}
		for _, p := range installed {
			status := "enabled"
			if !p.Enabled {
				status = "disabled"
			}
			fmt.Printf("  %-30s %-8s  %s\n", p.ID, p.Version, status)
		}
		return nil
	},
}

// --- plugin marketplace ---

var pluginMarketplaceCmd = &cobra.Command{
	Use:   "marketplace",
	Short: "Manage plugin marketplaces",
}

var pluginMarketplaceAddCmd = &cobra.Command{
	Use:   "add <source>",
	Short: "Add a plugin marketplace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]
		fmt.Printf("Adding marketplace %q... (not yet implemented in Go)\n", source)
		return nil
	},
}

var pluginMarketplaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured marketplaces",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Configured marketplaces:")
		fmt.Println("  (plugin marketplace system not yet implemented in Go)")
		return nil
	},
}

var pluginMarketplaceRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a marketplace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		fmt.Printf("Removing marketplace %q... (not yet implemented in Go)\n", name)
		return nil
	},
}

var pluginMarketplaceUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Update marketplace(s)",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Updating marketplaces... (not yet implemented in Go)")
		return nil
	},
}

// --- plugin install ---

var pluginInstallScope string

var pluginInstallCmd = &cobra.Command{
	Use:     "install <plugin>",
	Aliases: []string{"i"},
	Short:   "Install a plugin",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginRef := args[0]
		src, err := plugins.ResolveSource(pluginRef)
		if err != nil {
			return fmt.Errorf("invalid plugin reference %q: %w", pluginRef, err)
		}

		httpClient := &http.Client{Timeout: 60 * time.Second}
		cat := plugins.NewGitHubCatalog(httpClient)
		resolved, err := cat.Resolve(cmd.Context(), src, src.Repo)
		if err != nil {
			return fmt.Errorf("resolve plugin: %w", err)
		}
		defer resolved.Payload.Close()

		store := plugins.NewDiskStore()
		cacheDir, _ := pluginCacheDir()
		installed, err := store.Install(cmd.Context(), *resolved, cacheDir)
		if err != nil {
			return fmt.Errorf("install: %w", err)
		}

		// Update settings to enable the plugin
		pluginID := installed.Manifest.Name + "@" + pluginsMarketplaceName(src)
		if err := enablePluginInSettings(pluginID); err != nil {
			fmt.Println("Warning: installed but failed to enable in settings:", err)
		}

		fmt.Printf("Installed %s %s\n", installed.Manifest.Name, installed.Manifest.Version)
		return nil
	},
}

// --- plugin uninstall ---

var pluginUninstallScope string
var pluginUninstallKeepData bool

var pluginUninstallCmd = &cobra.Command{
	Use:     "uninstall <plugin>",
	Aliases: []string{"remove", "rm"},
	Short:   "Uninstall a plugin",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		store := plugins.NewDiskStore()
		cacheDir, err := pluginCacheDir()
		if err != nil {
			return err
		}
		return store.Uninstall(cmd.Context(), cacheDir, args[0])
	},
}

// --- plugin enable ---

var pluginEnableScope string

var pluginEnableCmd = &cobra.Command{
	Use:   "enable <plugin>",
	Short: "Enable a disabled plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		plugin := args[0]
		fmt.Printf("Enabling plugin %q... (not yet implemented in Go)\n", plugin)
		return nil
	},
}

// --- plugin disable ---

var pluginDisableScope string
var pluginDisableAll bool

var pluginDisableCmd = &cobra.Command{
	Use:   "disable [plugin]",
	Short: "Disable an enabled plugin (or --all)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if pluginDisableAll {
			fmt.Println("Disabling all plugins... (not yet implemented in Go)")
			return nil
		}
		if len(args) == 0 {
			return fmt.Errorf("specify a plugin name or use --all")
		}
		plugin := args[0]
		fmt.Printf("Disabling plugin %q... (not yet implemented in Go)\n", plugin)
		return nil
	},
}

// --- plugin update ---

var pluginUpdateScope string

var pluginUpdateCmd = &cobra.Command{
	Use:   "update <plugin>",
	Short: "Update a plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		plugin := args[0]
		fmt.Printf("Updating plugin %q... (not yet implemented in Go)\n", plugin)
		return nil
	},
}

func pluginCacheDir() (string, error) {
	if d := os.Getenv("CLAUDE_PLUGIN_CACHE_DIR"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".harness", "plugins", "cache"), nil
}

func pluginsMarketplaceName(source plugins.Source) string {
	if source.Repo != "" {
		parts := strings.Split(source.Repo, "/")
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return "local"
}

func enablePluginInSettings(pluginID string) error {
	// Simplified: write to user settings.go.json
	home, _ := os.UserHomeDir()
	settingsPath := filepath.Join(home, ".harness", "settings.go.json")

	var settings struct {
		EnabledPlugins map[string]bool `json:"enabledPlugins"`
	}
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		json.Unmarshal(data, &settings)
	}
	if settings.EnabledPlugins == nil {
		settings.EnabledPlugins = make(map[string]bool)
	}
	settings.EnabledPlugins[pluginID] = true

	newData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, newData, 0644)
}

func init() {
	// plugin list flags
	pluginListCmd.Flags().BoolVar(&pluginListJSON, "json", false, "Output as JSON")
	pluginListCmd.Flags().BoolVar(&pluginListAvailable, "available", false, "Show available plugins")

	// plugin install/uninstall/enable/disable/update flags
	pluginInstallCmd.Flags().StringVar(&pluginInstallScope, "scope", "", "Install scope")
	pluginUninstallCmd.Flags().StringVar(&pluginUninstallScope, "scope", "", "Uninstall scope")
	pluginUninstallCmd.Flags().BoolVar(&pluginUninstallKeepData, "keep-data", false, "Keep plugin data")
	pluginEnableCmd.Flags().StringVar(&pluginEnableScope, "scope", "", "Config scope")
	pluginDisableCmd.Flags().StringVar(&pluginDisableScope, "scope", "", "Config scope")
	pluginDisableCmd.Flags().BoolVar(&pluginDisableAll, "all", false, "Disable all plugins")
	pluginUpdateCmd.Flags().StringVar(&pluginUpdateScope, "scope", "", "Config scope")

	// marketplace subcommands
	pluginMarketplaceCmd.AddCommand(pluginMarketplaceAddCmd)
	pluginMarketplaceCmd.AddCommand(pluginMarketplaceListCmd)
	pluginMarketplaceCmd.AddCommand(pluginMarketplaceRemoveCmd)
	pluginMarketplaceCmd.AddCommand(pluginMarketplaceUpdateCmd)

	// plugin subcommands
	pluginCmd.AddCommand(pluginValidateCmd)
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginMarketplaceCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginUninstallCmd)
	pluginCmd.AddCommand(pluginEnableCmd)
	pluginCmd.AddCommand(pluginDisableCmd)
	pluginCmd.AddCommand(pluginUpdateCmd)
}
