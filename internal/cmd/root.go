package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AgentPal/AgentCRM/internal/store"
	"github.com/spf13/cobra"
)

var (
	configDir string
	format    string // "text" or "json"
	actor     string
)

// rootCmd 是 CLI 的根命令。
var rootCmd = &cobra.Command{
	Use:   "agentcrm",
	Short: "本地客户记忆系统 - 给你的 AI Agent 一个共享的客户大脑",
	Long: `AgentCRM 是一个本地化、文件存储、零服务进程的客户领域记忆系统。
它以 CLI 工具的形式分发，让 AI Agent 拥有一个长期、可读、可备份的客户记忆。

所有数据存储在 ~/AgentCRM/ 目录，文件即真相，可用任何编辑器打开。`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// 只有需要 store 的命令才初始化
		return nil
	},
}

// Execute 运行 CLI。
func Execute() error {
	return rootCmd.Execute()
}

// getStore 创建并返回 Store 实例。
func getStore() (*store.Store, error) {
	dir := configDir
	if dir == "" {
		dir = defaultDataDir()
	}
	return store.Open(dir)
}

// defaultDataDir 返回默认数据目录。
// 优先级：AGENTCRM_HOME 环境变量 > ~/AgentCRM
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

// getActor 返回 actor 值，优先级：--actor 参数 > AGENTCRM_ACTOR 环境变量 > "unknown"
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
	rootCmd.PersistentFlags().StringVar(&configDir, "data-dir", "", "数据目录（默认 ~/AgentCRM，可被 AGENTCRM_HOME 覆盖）")
	rootCmd.PersistentFlags().StringVar(&format, "format", "text", "输出格式: text 或 json")
	rootCmd.PersistentFlags().StringVar(&actor, "actor", "", "执行者名称（默认 AGENTCRM_ACTOR 环境变量或 unknown）")

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
	rootCmd.AddCommand(timelineCmd) // timeline 也可作为顶层命令使用
}

// appVersion 在构建时通过 -ldflags 注入。
var appVersion = "v1.0.0-dev"

// versionCmd 显示版本信息。
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("AgentCRM %s\n", appVersion)
		fmt.Println("本地客户记忆系统 - 文件即真相")
	},
}

// initCmd 初始化数据目录。
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化 AgentCRM 数据目录",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		if err := s.FS.Init(); err != nil {
			return fmt.Errorf("init directories: %w", err)
		}

		// 写入默认配置
		cfg, err := s.FS.ReadConfig()
		if err != nil {
			return err
		}
		if err := s.FS.WriteConfig(cfg); err != nil {
			return fmt.Errorf("write config: %w", err)
		}

		fmt.Printf("AgentCRM 已初始化: %s\n", s.ConfigDir())
		return nil
	},
}

// reindexCmd 从文件重建 SQLite 索引。
var reindexCmd = &cobra.Command{
	Use:   "reindex",
	Short: "从文件重建 SQLite 索引",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		return s.Reindex()
	},
}

