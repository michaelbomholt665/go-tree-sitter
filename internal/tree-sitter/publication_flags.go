package treesitter

import "github.com/spf13/cobra"

// Explicit flags override YAML, including explicit false values and legacy aliases.
func applyPublicationDefaults(cmd *cobra.Command, cfg *Config, flags map[string]*bool) {
	d := cfg.EffectiveRuntime().Publication
	values := map[string]bool{"manifest": BoolValue(d.Manifest, true), "compact": d.Compact, "check-compact": d.CheckCompact, "source": d.Source, "js": d.JS, "wasm": d.WASM, "force": d.Force, "allow-cross-validation": d.AllowCrossValidation, "clean": cfg.EffectiveRuntime().Cleanup.AfterMove != "keep"}
	aliases := map[string]string{"source": "c-source", "manifest": "no-manifest", "allow-cross-validation": "static-cross-validation"}
	for name, pointer := range flags {
		if cmd.Flags().Changed(name) {
			continue
		}
		if alias := aliases[name]; alias != "" && cmd.Flags().Changed(alias) {
			continue
		}
		*pointer = values[name]
	}
}
