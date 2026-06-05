package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/jeeftor/caddy-dns-sync/internal/tui"
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

type exitCodeError struct {
	Code int
	Err  error
}

func (e exitCodeError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit code %d", e.Code)
}

func exitCode(code int) error {
	return exitCodeError{Code: code}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "caddy-dns-sync",
	Short: "A CLI tool for managing Unbound DNS on OPNSense",
	Long: `
   ______          __    __      _____
  / ____/___ _____/ /___/ /_  __/ ___/__  ______  _____
 / /   / __ \/ __  / __  / / / /\__ \/ / / / __ \/ ___/
/ /___/ /_/ / /_/ / /_/ / /_/ /___/ / /_/ / / / / /__
\____/\__,_/\__,_/\__,_/\__, //____/\__, /_/ /_/\___/
                       /____/      /____/

A CLI tool for synchronizing DNS entries across Caddy, UnboundDNS, and AdguardHome.
This application manages DNS overrides on OPNSense routers and AdguardHome instances
by syncing hostname data from Caddy reverse proxy and Cloudflare tunnel configurations.`,
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
		var codeErr exitCodeError
		if errors.As(err, &codeErr) {
			if codeErr.Err != nil {
				fmt.Println(UI.RenderError(codeErr.Err))
			}
			os.Exit(codeErr.Code)
		}
		fmt.Println(UI.RenderError(err))
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", "config file (default is $HOME/.caddy-dns-sync.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().
		StringVar(&logLevel, "log-level", "info", "set logging level (debug, info, warn, error)")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))

	// Set version template
	rootCmd.SetVersionTemplate(
		`{{with .Name}}{{printf "%s " .}}{{end}}{{printf "version %s" .Version}}
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

		// Search config in home directory with name ".caddy-dns-sync" (without extension)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".caddy-dns-sync")
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
