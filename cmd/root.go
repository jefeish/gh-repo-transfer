package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	outputFormat string
	verbose      bool
	sections     []string
	targetOrg    string
	separateFiles bool
	dryRun       bool
	enforce      bool
	assign       bool
	createTeams  bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "repo-transfer [command] [repos...] [flags]",
	Short: "GitHub CLI extension for discovering repository governance and organizational dependencies",
	Long: `gh repo-transfer is a GitHub CLI extension for discovering 
repository governance configuration and organizational dependencies.

This tool can perform two types of analysis:
1. Governance inspection (rulesets, collaborators, security settings, etc.)
2. Organizational dependencies analysis (code deps, CI/CD deps, access control, etc.)`,
	RunE: runInspect,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Disable the completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	
	// Set custom help template to match desired order
	rootCmd.SetHelpTemplate(`{{.Long | trimTrailingWhitespaces}}

Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

Examples:
  repo-transfer deps owner/repo                                  # Analyze single repository
  repo-transfer deps owner/repo1 owner/repo2 owner/repo3         # Batch analysis
  repo-transfer deps owner/repo --target-org target-org          # With automatic validation
  repo-transfer deps owner/repo1 owner/repo2 --per-repo          # Output to individual files
  repo-transfer transfer owner/repo --target-org org             # Transfer repository
  repo-transfer transfer owner/repo --target-org org --dry-run   # Preview transfer
  repo-transfer transfer owner/repo --target-org org --enforce   # Enforce transfer despite validation blockers
  repo-transfer transfer owner/repo --target-org org --assign    # Transfer and assign to same teams

{{if .HasAvailableSubCommands}}Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`)
	
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "table", "Output format (json, yaml, table)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVarP(&targetOrg, "target-org", "t", "", "Target organization for validation or transfer")
	rootCmd.PersistentFlags().BoolVarP(&separateFiles, "per-repo", "p", false, "Output analysis to individual JSON files (deps only)")
	rootCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "d", false, "Preview actions without executing (transfer only)")
	rootCmd.PersistentFlags().BoolVarP(&enforce, "enforce", "e", false, "Enforce transfer action even if validation shows blockers (transfer only)")
	rootCmd.PersistentFlags().BoolVarP(&assign, "assign", "a", false, "Apply existing teams after repository transfer (transfer only)")
	rootCmd.PersistentFlags().BoolVarP(&createTeams, "create", "c", false, "Create teams in target org if they don't exist (transfer/archive only)")
	rootCmd.Flags().StringSliceVarP(&sections, "sections", "s", nil, "Specific sections to inspect \n(rulesets, collaborators, teams, security, settings, labels, milestones)")
}

func runInspect(cmd *cobra.Command, args []string) error {
	// Show help when no subcommand is provided
	return cmd.Help()
}