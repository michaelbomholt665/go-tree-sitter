package treesitter

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

func selectSettingAction(message string, options []string, defaults ...string) (string, error) {
	var answer string
	prompt := &survey.Select{Message: message, Options: options}
	if len(defaults) > 0 {
		prompt.Default = defaults[0]
	}
	err := survey.AskOne(prompt, &answer)
	return answer, err
}

func settingCategory(key string) string {
	parts := strings.Split(key, ".")
	if parts[0] == "runtime" {
		return "Runtime: " + strings.ReplaceAll(parts[1], "_", " ")
	}
	if parts[0] == "output" {
		return "Output and publication paths"
	}
	if parts[0] == "targets" {
		return "Target toolchains"
	}
	return "Build and ABI"
}

func settingSummary(value any) string {
	data, _ := json.Marshal(value)
	text := string(data)
	if len(text) > 85 {
		text = text[:82] + "..."
	}
	return text
}

func (a *App) editSettings(cmd *cobra.Command, path string) error {
	doc, err := readCommandConfigDocument(cmd, path)
	if err != nil {
		return err
	}
	original := doc.cfg
	changes := map[string]any{}
	categories := map[string][]configSetting{}
	for _, setting := range configSettings() {
		group := settingCategory(setting.key)
		categories[group] = append(categories[group], setting)
	}
	groups := make([]string, 0, len(categories))
	for group := range categories {
		groups = append(groups, group)
	}
	sort.Strings(groups)
	fmt.Fprintln(a.stdout, "Settings:", path, "\nChanges remain in memory until you review and save. Grammar pins and evidence are not edited here.")
	if isDryRun(cmd) {
		fmt.Fprintln(a.stdout, "Dry run: confirmation previews saving; settings will not be written.")
	}
	for {
		options := append(append([]string{}, groups...), "Review and save", "Discard and return")
		group, err := selectSettingAction("Settings section:", options)
		if err != nil {
			return err
		}
		switch group {
		case "Discard and return":
			return nil
		case "Review and save":
			if len(changes) == 0 {
				fmt.Fprintln(a.stdout, "No pending changes.")
				continue
			}
			if err := writeConfigValue(a.stdout, changes, false); err != nil {
				return err
			}
			save := false
			if err := survey.AskOne(&survey.Confirm{Message: "Save these settings to " + path + "?", Default: false}, &save); err != nil {
				return err
			}
			if !save {
				continue
			}
			if err := cmd.Context().Err(); err != nil {
				return err
			}
			if isDryRun(cmd) {
				fmt.Fprintln(a.stdout, "Dry run: settings validated; would save to", path, "(not saved).")
				return nil
			}
			if err := doc.save(); err != nil {
				return err
			}
			fmt.Fprintln(a.stdout, "Settings saved.")
			return nil
		}
		settings := categories[group]
		labels := []string{}
		for _, setting := range settings {
			value, err := configValue(doc.cfg, setting.key)
			if err != nil {
				return err
			}
			labels = append(labels, setting.key+" = "+settingSummary(value))
		}
		labels = append(labels, "Back")
		selected, err := selectSettingAction(group+":", labels)
		if err != nil {
			return err
		}
		if selected == "Back" {
			continue
		}
		var setting configSetting
		for i, label := range labels[:len(labels)-1] {
			if label == selected {
				setting = settings[i]
				break
			}
		}
		current, err := configValue(doc.cfg, setting.key)
		if err != nil {
			return err
		}
		fmt.Fprintln(a.stdout, settingHelp(setting.key))
		value, err := editSettingValue(setting.key, setting.valueType, current, doc.cfg)
		if err != nil {
			return err
		}
		input, err := json.Marshal(value)
		if err != nil {
			return err
		}
		if err := doc.set(setting.key, string(input)); err != nil {
			fmt.Fprintln(a.stdout, "Not changed:", err)
			continue
		}
		before, err := configValue(original, setting.key)
		if err != nil {
			return err
		}
		after, err := configValue(doc.cfg, setting.key)
		if err != nil {
			return err
		}
		if reflect.DeepEqual(before, after) && value != nil {
			delete(changes, setting.key)
		} else {
			changes[setting.key] = map[string]any{"before": before, "after": after, "warnings": settingWarnings(setting.key, after)}
		}
		for _, warning := range settingWarnings(setting.key, after) {
			fmt.Fprintln(a.stdout, "WARNING:", warning)
		}
	}
}

func editSettingValue(key string, typ reflect.Type, current any, cfg *Config) (any, error) {
	if typ.Kind() == reflect.Pointer && typ.Elem().Kind() == reflect.Bool {
		options := []string{"true", "false", "null (use default)"}
		initial := fmt.Sprint(current)
		if current == nil {
			initial = options[2]
		}
		selected, err := selectSettingAction(key+" (current "+settingSummary(current)+"):", options, initial)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(selected, "null") {
			return nil, nil
		}
		return selected == "true", nil
	}
	typ = settingType(typ)
	if choices := settingChoices(key, cfg); len(choices) > 0 {
		return selectSettingAction(key+" (current "+settingSummary(current)+"):", choices, fmt.Sprint(current))
	}
	switch typ.Kind() {
	case reflect.Bool:
		value, _ := current.(bool)
		err := survey.AskOne(&survey.Confirm{Message: key + ":", Default: value}, &value)
		return value, err
	case reflect.String:
		value, _ := current.(string)
		err := survey.AskOne(&survey.Input{Message: key + ":", Default: value}, &value)
		return value, err
	case reflect.Int:
		input := fmt.Sprint(current)
		if current == nil {
			input = "0"
		}
		err := survey.AskOne(&survey.Input{Message: key + " (integer):", Default: input}, &input)
		if err != nil {
			return nil, err
		}
		var value int
		if err := json.Unmarshal([]byte(input), &value); err != nil {
			return nil, err
		}
		return value, nil
	case reflect.Struct:
		value, _ := current.(map[string]any)
		if value == nil {
			value = map[string]any{}
		}
		fields := []string{}
		types := map[string]reflect.Type{}
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			name := strings.Split(field.Tag.Get("yaml"), ",")[0]
			if name == "" || name == "-" {
				continue
			}
			fields = append(fields, name)
			types[name] = field.Type
		}
		for {
			field, err := selectSettingAction(key+" fields:", append(append([]string{}, fields...), "Done"))
			if err != nil {
				return nil, err
			}
			if field == "Done" {
				return value, nil
			}
			changed, err := editSettingValue(key+"."+field, types[field], value[field], cfg)
			if err != nil {
				return nil, err
			}
			value[field] = changed
		}
	case reflect.Map:
		value, _ := current.(map[string]any)
		if value == nil {
			value = map[string]any{}
		}
		for {
			action, err := selectSettingAction(key+" ("+settingSummary(value)+"):", []string{"Add entry", "Edit entry", "Remove entry", "Done"})
			if err != nil {
				return nil, err
			}
			if action == "Done" {
				return value, nil
			}
			entry := ""
			if action == "Add entry" {
				if err := survey.AskOne(&survey.Input{Message: "New entry name:"}, &entry, survey.WithValidator(survey.Required)); err != nil {
					return nil, err
				}
				if _, exists := value[entry]; exists {
					fmt.Println("That entry exists; use Edit entry.")
					continue
				}
			} else {
				keys := []string{}
				for name := range value {
					keys = append(keys, name)
				}
				sort.Strings(keys)
				if len(keys) == 0 {
					continue
				}
				entry, err = selectSettingAction("Entry:", keys)
				if err != nil {
					return nil, err
				}
			}
			if action == "Remove entry" {
				delete(value, entry)
				continue
			}
			changed, err := editSettingValue(key+"."+entry, typ.Elem(), value[entry], cfg)
			if err != nil {
				return nil, err
			}
			value[entry] = changed
		}
	case reflect.Slice:
		value, _ := current.([]any)
		if value == nil {
			value = []any{}
		}
		for {
			action, err := selectSettingAction(key+" ("+settingSummary(value)+"):", []string{"Add item", "Edit item", "Remove item", "Done"})
			if err != nil {
				return nil, err
			}
			if action == "Done" {
				return value, nil
			}
			index := len(value)
			if action != "Add item" {
				labels := []string{}
				for i, item := range value {
					labels = append(labels, fmt.Sprintf("%d: %s", i+1, settingSummary(item)))
				}
				if len(labels) == 0 {
					continue
				}
				item, err := selectSettingAction("Item:", labels)
				if err != nil {
					return nil, err
				}
				for i, label := range labels {
					if label == item {
						index = i
						break
					}
				}
			}
			if action == "Remove item" {
				value = append(value[:index], value[index+1:]...)
				continue
			}
			var previous any
			if index < len(value) {
				previous = value[index]
			}
			changed, err := editSettingValue(key+" item", typ.Elem(), previous, cfg)
			if err != nil {
				return nil, err
			}
			if index == len(value) {
				value = append(value, changed)
			} else {
				value[index] = changed
			}
		}
	}
	return nil, fmt.Errorf("unsupported setting type %s", typ)
}

func settingHelp(key string) string {
	switch {
	case key == "build_dir":
		return "Build workspace. Relative paths resolve from the selected configuration directory."
	case strings.HasPrefix(key, "output."):
		return "Publication paths and defaults. Build/compile templates support {build_dir}, {lang}, {os}, and {arch}."
	case strings.HasPrefix(key, "runtime.cleanup."):
		return "Controls workspace retention after successful publication and patterns used during pruning."
	case strings.HasPrefix(key, "runtime.tools."):
		return "Executable name on PATH or explicit executable path; not a shell command."
	case strings.HasPrefix(key, "runtime.node."):
		return "Node dependency policy. Lifecycle scripts remain disabled by validation."
	case strings.HasPrefix(key, "runtime.queries."):
		return "Repository-relative query paths and per-language text rewrites."
	case strings.HasPrefix(key, "runtime.sync_modules"):
		return "Go module names map to grammar identifiers used by sync-config."
	case strings.HasPrefix(key, "runtime.wizard."):
		return "Default selections for future interactive builds; these do not start a build."
	case strings.HasPrefix(key, "runtime.publication."):
		return "Default publication options; explicit command flags override them."
	case strings.Contains(key, "abi"):
		return "Parser ABI settings are not source-language version claims."
	case key == "targets" || strings.HasPrefix(key, "runtime.compiler."):
		return "Target compiler paths, version pins, and fallback policy. Use trusted executables."
	case strings.HasPrefix(key, "runtime.wasm."):
		return "WASM backend and container image. Use a trusted, pinned image for reproducibility."
	default:
		return "Configuration value; the complete configuration must remain valid."
	}
}

func (a *App) wizardSettingsMenu(cmd *cobra.Command, path string) (bool, error) {
	for {
		action, err := selectSettingAction("ts-build:", []string{"Build grammars", "Settings", "Exit"})
		if err != nil {
			return false, err
		}
		switch action {
		case "Build grammars":
			return true, nil
		case "Exit":
			return false, nil
		case "Settings":
			if err := a.editSettings(cmd, path); err != nil {
				return false, err
			}
		}
	}
}
