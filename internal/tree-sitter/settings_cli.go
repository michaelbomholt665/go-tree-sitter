package treesitter

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/cobra"
)

func (a *App) addSettingsCommands(parent *cobra.Command, configPath *string) {
	for _, mode := range []string{"show", "get", "validate"} {
		var jsonOutput bool
		cmd := &cobra.Command{Use: mode, Args: cobra.NoArgs}
		switch mode {
		case "show":
			cmd.Short = "Show non-language settings with defaults applied"
		case "get":
			cmd.Use = "get <key>"
			cmd.Short = "Read one non-language setting"
			cmd.Args = cobra.ExactArgs(1)
		case "validate":
			cmd.Short = "Validate configuration and workspace separation"
		}
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			doc, err := readCommandConfigDocument(cmd, *configPath)
			if err != nil {
				return err
			}
			switch mode {
			case "get":
				if _, err := findConfigSetting(args[0]); err != nil {
					return err
				}
				value, err := configValue(doc.cfg, args[0])
				if err != nil {
					return err
				}
				return writeConfigValue(a.stdout, value, jsonOutput)
			case "validate":
				if jsonOutput {
					return writeConfigValue(a.stdout, map[string]any{"valid": true, "config": *configPath}, true)
				}
				_, err := fmt.Fprintln(a.stdout, "Configuration is valid:", *configPath)
				return err
			default:
				values, err := settingsValues(doc.cfg)
				if err != nil {
					return err
				}
				return writeConfigValue(a.stdout, values, jsonOutput)
			}
		}
		cmd.Flags().BoolVar(&jsonOutput, "json", false, "Emit JSON output")
		parent.AddCommand(cmd)
	}
	var apply, jsonOutput bool
	set := &cobra.Command{Use: "set <key> <value>", Short: "Preview a non-language setting change; use --apply to save", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer func() {
			if err != nil && jsonOutput {
				_ = json.NewEncoder(a.stdout).Encode(map[string]string{"error": err.Error()})
			}
		}()
		if apply && isDryRun(cmd) {
			return errors.New("--apply and --dry-run are mutually exclusive")
		}
		doc, err := readCommandConfigDocument(cmd, *configPath)
		if err != nil {
			return err
		}
		if _, err := findConfigSetting(args[0]); err != nil {
			return err
		}
		before, err := configValue(doc.cfg, args[0])
		if err != nil {
			return err
		}
		previous, err := encodeConfigNode(&doc.node)
		if err != nil {
			return err
		}
		if err := doc.set(args[0], args[1]); err != nil {
			return err
		}
		next, err := encodeConfigNode(&doc.node)
		if err != nil {
			return err
		}
		changed := string(previous) != string(next)
		after, err := configValue(doc.cfg, args[0])
		if err != nil {
			return err
		}
		warnings := settingWarnings(args[0], after)
		if apply && changed {
			if err := cmd.Context().Err(); err != nil {
				return err
			}
			if err := doc.save(); err != nil {
				return err
			}
		}
		result := map[string]any{"key": args[0], "before": before, "after": after, "changed": changed, "applied": apply, "warnings": warnings}
		if jsonOutput {
			return writeConfigValue(a.stdout, result, true)
		}
		status := "Preview (not saved)"
		if apply {
			status = "Applied"
		}
		if _, err := fmt.Fprintln(a.stdout, status); err != nil {
			return err
		}
		return writeConfigValue(a.stdout, result, false)
	}}
	set.Flags().BoolVar(&apply, "apply", false, "Save the validated setting change")
	set.Flags().BoolVar(&jsonOutput, "json", false, "Emit structured results and setting validation errors")
	parent.AddCommand(set, &cobra.Command{Use: "edit", Short: "Interactively edit non-language settings, review, then save", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error { return a.editSettings(cmd, *configPath) }})
}

func settingsValues(cfg *Config) (map[string]any, error) {
	var values = make(map[string]any)
	for _, setting := range configSettings() {
		value, err := configValue(cfg, setting.key)
		if err != nil {
			return nil, err
		}
		current := values
		parts := strings.Split(setting.key, ".")
		for _, part := range parts[:len(parts)-1] {
			next, ok := current[part].(map[string]any)
			if !ok {
				next = make(map[string]any)
				current[part] = next
			}
			current = next
		}
		current[parts[len(parts)-1]] = value
	}
	return values, nil
}

func settingWarnings(key string, value any) []string {
	warnings := []string{}
	if key == "runtime.cleanup.after_move" && value == "remove" {
		warnings = append(warnings, "Successful publication removes the grammar workspace; failed publication preserves it.")
	}
	if strings.HasPrefix(key, "runtime.cleanup.prune.") {
		warnings = append(warnings, "Pruning can delete matching workspace files after publication or during clean --prune.")
	}
	if strings.HasPrefix(key, "runtime.tools.") || strings.HasPrefix(key, "runtime.compiler.") || key == "targets" || key == "runtime.wasm.image" {
		warnings = append(warnings, "Future builds execute the configured toolchain or container image. Use trusted programs and images.")
	}
	return warnings
}

func settingChoices(key string, cfg *Config) []string {
	switch key {
	case "runtime.cleanup.after_move":
		return []string{"keep", "prune", "remove"}
	case "runtime.wasm.backend":
		return []string{"auto", "emcc", "docker", "podman"}
	case "runtime.node.install":
		return []string{"never", "auto", "always"}
	case "output.default_move_mode":
		return cfg.Output.SupportedMoveModes
	case "OS_TARGET":
		return []string{"linux", "macos", "windows"}
	}
	return nil
}

func settingType(typ reflect.Type) reflect.Type {
	if typ.Kind() == reflect.Pointer {
		return typ.Elem()
	}
	return typ
}
