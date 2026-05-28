package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AgentPal/AgentCRM/internal/i18n"
	"github.com/AgentPal/AgentCRM/internal/store"
	"github.com/spf13/cobra"
)

var (
	configDir string
	format    string // "text" or "json"
	actor     string
)

// rootCmd is the root command of the CLI.
var rootCmd = &cobra.Command{
	SilenceErrors:  true,
	SilenceUsage:   true,
	Use:            "agentcrm",
	Short:          i18n.T("cmd.root.short"),
	Long:           i18n.T("cmd.root.long"),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

// getStore creates and returns a Store instance.
func getStore() (*store.Store, error) {
	dir := configDir
	if dir == "" {
		dir = defaultDataDir()
	}
	return store.Open(dir)
}

// defaultDataDir returns the default data directory.
// Priority: AGENTCRM_HOME env > ~/AgentCRM
func defaultDataDir() string {
	if env := os.Getenv("AGENTCRM_HOME"); env != "" {
		return env
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "./AgentCRM"
	}
	return filepath.Join(home, "AgentCRM")
}

// getActor returns the actor value.
// Priority: --actor flag > AGENTCRM_ACTOR env > "unknown"
func getActor() string {
	if actor != "" {
		return actor
	}
	if env := os.Getenv("AGENTCRM_ACTOR"); env != "" {
		return env
	}
	return "unknown"
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configDir, "data-dir", "", i18n.T("flag.data_dir"))
	rootCmd.PersistentFlags().StringVar(&format, "format", "text", i18n.T("flag.format"))
	rootCmd.PersistentFlags().StringVar(&actor, "actor", "", i18n.T("flag.actor"))

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(reindexCmd)
	rootCmd.AddCommand(doctorCmd)

	rootCmd.AddCommand(contactCmd)
	rootCmd.AddCommand(dealCmd)
	rootCmd.AddCommand(activityCmd)
	rootCmd.AddCommand(memoryCmd)
	rootCmd.AddCommand(eventsCmd)
	rootCmd.AddCommand(alertCmd)
	rootCmd.AddCommand(ruleCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(timelineCmd)
}

// appVersion is injected at build time via -ldflags.
var appVersion = "v1.0.0-dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: i18n.T("cmd.root.version"),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("AgentCRM %s\n", appVersion)
		fmt.Println(i18n.T("output.root.version.text"))
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: i18n.T("cmd.root.init"),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		if err := s.FS.Init(); err != nil {
			return fmt.Errorf("init directories: %w", err)
		}

		cfg, err := s.FS.ReadConfig()
		if err != nil {
			return err
		}
		if err := s.FS.WriteConfig(cfg); err != nil {
			return fmt.Errorf("write config: %w", err)
		}

		fmt.Printf(i18n.T("output.root.init.success")+"\n", s.ConfigDir())
		return nil
	},
}

var reindexCmd = &cobra.Command{
	Use:   "reindex",
	Short: i18n.T("cmd.root.reindex"),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		return s.Reindex()
	},
}
