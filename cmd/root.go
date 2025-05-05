package cmd

import (
	"fmt"
	"os"

	"github.com/jeeftor/unboundCLI/internal/logging"
	"github.com/jeeftor/unboundCLI/internal/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Version information (set by build using ldflags)
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

var (
	cfgFile  string
	verbose  bool
	logLevel string

	// Global UI instance for consistent styling
	UI = tui.NewUI()
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "unboundCLI",
	Short: "A CLI tool for managing Unbound DNS on OPNSense",
	Long: `
u2588u2588u2557   u2588u2588u2557u2588u2588u2588u2557   u2588u2588u2557u2588u2588u2588u2588u2588u2588u2557  u2588u2588u2588u2588u2588u2588u2557 u2588u2588u2557   u2588u2588u2557u2588u2588u2588u2557   u2588u2588u2557u2588u2588u2588u2588u2588u2588u2557  u2588u2588u2588u2588u2588u2588u2557u2588u2588u2557     u2588u2588u2557
u2588u2588u2551   u2588u2588u2551u2588u2588u2588u2588u2557  u2588u2588u2551u2588u2588u2554u2550u2550u2588u2588u2557u2588u2588u2554u2550u2550u2550u2588u2588u2557u2588u2588u2551   u2588u2588u2551u2588u2588u2588u2588u2557  u2588u2588u2551u2588u2588u2554u2550u2550u2588u2588u2557u2588u2588u2554u2550u2550u2550u2550u255du2588u2588u2551     u2588u2588u2551
u2588u2588u2551   u2588u2588u2551u2588u2588u2554u2588u2588u2557 u2588u2588u2551u2588u2588u2588u2588u2588u2588u2554u255du2588u2588u2551   u2588u2588u2551u2588u2588u2551   u2588u2588u2551u2588u2588u2554u2588u2588u2557 u2588u2588u2551u2588u2588u2551  u2588u2588u2551u2588u2588u2551     u2588u2588u2551     u2588u2588u2551
u2588u2588u2551   u2588u2588u2551u2588u2588u2551u255au2588u2588u2557u2588u2588u2551u2588u2588u2554u2550u2550u2588u2588u2557u2588u2588u2551   u2588u2588u2551u2588u2588u2551   u2588u2588u2551u2588u2588u2551u255au2588u2588u2557u2588u2588u2551u2588u2588u2551  u2588u2588u2551u2588u2588u2551     u2588u2588u2551     u2588u2588u2551
u255au2588u2588u2588u2588u2588u2588u2554u255du2588u2588u2551 u255au2588u2588u2588u2588u2551u2588u2588u2588u2588u2588u2588u2554u255du255au2588u2588u2588u2588u2588u2588u2554u255du255au2588u2588u2588u2588u2588u2588u2554u255du2588u2588u2551 u255au2588u2588u2588u2588u2551u2588u2588u2588u2588u2588u2588u2554u255du255au2588u2588u2588u2588u2588u2588u2557u2588u2588u2588u2588u2588u2588u2588u2557u2588u2588u2551
 u255au2550u2550u2550u2550u2550u255d u255au2550u255d  u255au2550u2550u2550u255du255au2550u2550u2550u2550u2550u255d  u255au2550u2550u2550u2550u2550u255d  u255au2550u2550u2550u2550u2550u255d u255au2550u255d  u255au2550u2550u2550u255du255au2550u2550u2550u2550u2550u255d  u255au2550u2550u2550u2550u2550u255du255au2550u2550u2550u2550u2550u2550u255du255au2550u255d

A CLI tool for managing Unbound DNS on OPNSense routers.
This application allows you to create, read, update, and delete DNS overrides
through the OPNSense API.`,
	Version: Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize logger with the specified log level
		var level logging.LogLevel
		switch logLevel {
		case "debug":
			level = logging.LogLevelDebug
		case "info":
			level = logging.LogLevelInfo
		case "warn":
			level = logging.LogLevelWarn
		case "error":
			level = logging.LogLevelError
		default:
			level = logging.LogLevelInfo
		}
		logging.Init(level)
		logging.Debug("Logging initialized", "level", logLevel)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Println(UI.RenderError(err))
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", "config file (default is $HOME/.unboundCLI.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().
		StringVar(&logLevel, "log-level", "info", "set logging level (debug, info, warn, error)")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))

	// Set version template
	rootCmd.SetVersionTemplate(
		`{{with .Name}}{{printf "%s " .}}{{end}}{{printf "version %s" .Version}}
Commit: {{printf "%s" .Commit}}
Built: {{printf "%s" .Date}}
`,
	)
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".unboundCLI" (without extension)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".unboundCLI")
	}

	// Read in environment variables that match
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Println(UI.RenderInfo("Using config file: " + viper.ConfigFileUsed()))
		}
	}

	// Set default values
	viper.SetDefault("insecure", false)
}
