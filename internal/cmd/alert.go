package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/AgentPal/AgentCRM/internal/alert"
	"github.com/AgentPal/AgentCRM/internal/model"
	"github.com/AgentPal/AgentCRM/internal/store"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// alertCmd 管理提醒。
var alertCmd = &cobra.Command{
	Use:   "alert",
	Short: "管理提醒",
}

var alertScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "扫描并生成提醒",
	RunE: func(cmd *cobra.Command, args []string) error {
		sinceLast, _ := cmd.Flags().GetBool("since-last-scan")

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		// 读取现有提醒
		existing, _ := s.FS.ReadPendingAlerts()

		// 过滤掉已关闭的
		var active []model.Alert
		for _, a := range existing {
			if a.Status == model.AlertStatusPending {
				active = append(active, a)
			}
		}

		if sinceLast && len(active) > 0 {
			fmt.Printf("存在 %d 条未处理提醒，跳过扫描\n", len(active))
			return nil
		}

		engine := alert.NewEngine(s, s.ConfigDir())
		alerts, err := engine.Scan()
		if err != nil {
			return fmt.Errorf("scan alerts: %w", err)
		}

		// 合并现有 + 新提醒
		all := append(active, alerts...)
		if err := s.FS.WritePendingAlerts(all); err != nil {
			return fmt.Errorf("save alerts: %w", err)
		}

		if format == "json" {
			out, _ := json.Marshal(alerts)
			fmt.Println(string(out))
		} else {
			if len(alerts) == 0 {
				fmt.Println("未发现新问题")
				return nil
			}
			for _, a := range alerts {
				fmt.Printf("[%s] %s\n", a.RuleName, a.Title)
				if a.Suggestion != "" {
					fmt.Printf("  建议: %s\n", a.Suggestion)
				}
				if a.ContactName != "" {
					fmt.Printf("  联系人: %s\n", a.ContactName)
				}
				if a.DealTitle != "" {
					fmt.Printf("  商机: %s\n", a.DealTitle)
				}
			}
			fmt.Printf("\n发现 %d 个新提醒\n", len(alerts))
		}
		return nil
	},
}

var alertListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出当前提醒",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		alerts, err := s.FS.ReadPendingAlerts()
		if err != nil {
			return err
		}

		var pending []model.Alert
		for _, a := range alerts {
			if a.Status == model.AlertStatusPending {
				pending = append(pending, a)
			}
		}

		if format == "json" {
			out, _ := json.Marshal(pending)
			fmt.Println(string(out))
		} else {
			if len(pending) == 0 {
				fmt.Println("无待处理提醒")
				return nil
			}
			for _, a := range pending {
				fmt.Printf("[%s] %s (%s)\n", a.RuleName, a.Title, a.ID)
				if a.Suggestion != "" {
					fmt.Printf("  建议: %s\n", a.Suggestion)
				}
			}
			fmt.Printf("\n共 %d 条待处理提醒\n", len(pending))
		}
		return nil
	},
}

var alertDismissCmd = &cobra.Command{
	Use:   "dismiss <id>",
	Short: "忽略一条提醒",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		permanent, _ := cmd.Flags().GetBool("permanent")
		reason, _ := cmd.Flags().GetString("reason")

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		alerts, err := s.FS.ReadPendingAlerts()
		if err != nil {
			return err
		}

		found := false
		now := store.Now()
		for i := range alerts {
			if alerts[i].ID == args[0] {
				alerts[i].Status = model.AlertStatusDismissed
				alerts[i].DismissedAt = now
				alerts[i].DismissReason = reason
				alerts[i].DismissPermanent = permanent
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("提醒未找到: %s", args[0])
		}

		if err := s.FS.WritePendingAlerts(alerts); err != nil {
			return err
		}
		fmt.Println("已忽略提醒")
		return nil
	},
}

var alertSnoozeCmd = &cobra.Command{
	Use:   "snooze <id>",
	Short: "推迟提醒",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		days, _ := cmd.Flags().GetInt("days")

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		alerts, err := s.FS.ReadPendingAlerts()
		if err != nil {
			return err
		}

		found := false
		now := time.Now().UTC()
		for i := range alerts {
			if alerts[i].ID == args[0] {
				alerts[i].Status = model.AlertStatusSnoozed
				alerts[i].SnoozedUntil = now.AddDate(0, 0, days).Format(time.RFC3339)
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("提醒未找到: %s", args[0])
		}

		if err := s.FS.WritePendingAlerts(alerts); err != nil {
			return err
		}
		fmt.Printf("已推迟提醒 %d 天\n", days)
		return nil
	},
}

// ruleCmd 管理规则。
var ruleCmd = &cobra.Command{
	Use:   "rule",
	Short: "管理提醒规则",
}

var ruleListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出规则",
	RunE: func(cmd *cobra.Command, args []string) error {
		rules := alert.BuiltinRules()

		if format == "json" {
			out, _ := json.Marshal(rules)
			fmt.Println(string(out))
		} else {
			fmt.Println("内置规则:")
			for _, r := range rules {
				fmt.Printf("  %s: %s\n", r.Name, r.Then.Alert.Title)
			}

			// 尝试读取用户规则
			s, err := getStore()
			if err == nil {
				data, err := os.ReadFile(filepath.Join(s.ConfigDir(), "rules", "user.yaml"))
				if err == nil {
					var cfg model.RulesConfig
					if yaml.Unmarshal(data, &cfg) == nil && len(cfg.Rules) > 0 {
						fmt.Println("\n用户规则:")
						for _, r := range cfg.Rules {
							fmt.Printf("  %s: %s\n", r.Name, r.Then.Alert.Title)
						}
					}
				}
			}
			s.Close()
		}
		return nil
	},
}

var ruleAddCmd = &cobra.Command{
	Use:   "add",
	Short: "添加规则",
	RunE: func(cmd *cobra.Command, args []string) error {
		fromYAML, _ := cmd.Flags().GetString("from-yaml")

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		data, err := os.ReadFile(fromYAML)
		if err != nil {
			return fmt.Errorf("read yaml: %w", err)
		}

		// 验证格式
		var cfg model.RulesConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("parse yaml: %w", err)
		}
		if len(cfg.Rules) == 0 {
			return fmt.Errorf("规则文件为空")
		}

		userPath := filepath.Join(s.ConfigDir(), "rules", "user.yaml")
		if err := os.MkdirAll(filepath.Dir(userPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(userPath, data, 0644); err != nil {
			return fmt.Errorf("write user rules: %w", err)
		}

		fmt.Printf("已添加 %d 条规则\n", len(cfg.Rules))
		return nil
	},
}

var ruleDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "禁用规则",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// 禁用方式：添加 disabled 规则到 user.yaml
		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		userPath := filepath.Join(s.ConfigDir(), "rules", "disabled.yaml")
		// 追加规则名到禁用列表
		f, err := os.OpenFile(userPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		fmt.Fprintf(f, "%s\n", args[0])

		fmt.Printf("已禁用规则: %s\n", args[0])
		return nil
	},
}

func init() {
	alertCmd.AddCommand(alertScanCmd)
	alertCmd.AddCommand(alertListCmd)
	alertCmd.AddCommand(alertDismissCmd)
	alertCmd.AddCommand(alertSnoozeCmd)

	ruleCmd.AddCommand(ruleListCmd)
	ruleCmd.AddCommand(ruleAddCmd)
	ruleCmd.AddCommand(ruleDisableCmd)

	alertScanCmd.Flags().Bool("since-last-scan", false, "仅上次扫描后的新数据")
	alertDismissCmd.Flags().Bool("permanent", false, "永久忽略")
	alertDismissCmd.Flags().String("reason", "", "忽略原因")
	alertSnoozeCmd.Flags().Int("days", 7, "推迟天数")

	ruleAddCmd.Flags().String("from-yaml", "", "YAML 文件路径")
}
