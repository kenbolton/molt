package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "molt",
	Short: "Migrate claw agent installations between architectures",
	Long: `molt — portable migration for claw agent architectures.

Move your agents, groups, memory, skills, and config between
NanoClaw, OpenClaw, ZeptoClaw, PicoClaw, and others.

Examples:
  molt export ~/nanoclaw-install --out my-agents.molt
  molt import my-agents.molt ~/new-install --arch zepto
  molt ~/old-install ~/new-install --arch zepto`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 2 {
			// Combined export+import shorthand: molt <source> <dest> --arch <name>
			return runCombined(cmd, args[0], args[1])
		}
		return cmd.Help()
	},
}

var (
	flagArch   string
	flagRename []string
	flagDryRun bool
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagArch, "arch", "", "Target architecture (e.g. nanoclaw, zepto, openclaw, pico)")
	rootCmd.PersistentFlags().StringArrayVar(&flagRename, "rename", nil, "Rename group slug on import: --rename old=new (repeatable)")
	rootCmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "Show what would happen without making changes")

	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(archsCmd)
}

func runCombined(cmd *cobra.Command, source, dest string) error {
	if flagArch == "" {
		return fmt.Errorf("--arch is required\n\nRe-run with:\n  molt %s %s --arch <nanoclaw|zepto|openclaw|pico>", source, dest)
	}
	// Export to temp bundle, then import
	// TODO: implement
	return fmt.Errorf("combined molt not yet implemented — use export + import separately")
}
