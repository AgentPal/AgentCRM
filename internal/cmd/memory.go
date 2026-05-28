package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/agentcrm/agentcrm/internal/model"
	"github.com/agentcrm/agentcrm/internal/search"
	"github.com/spf13/cobra"
)

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "管理长期记忆",
}

var memoryWriteCmd = &cobra.Command{
	Use:   "write",
	Short: "写入记忆",
	RunE: func(cmd *cobra.Command, args []string) error {
		scope, _ := cmd.Flags().GetString("scope")
		text, _ := cmd.Flags().GetString("text")
		decay, _ := cmd.Flags().GetString("decay")
		validFrom, _ := cmd.Flags().GetString("valid-from")

		if scope == "" || text == "" {
			return fmt.Errorf("--scope 和 --text 是必需的")
		}

		scopeType, scopeID := parseScope(scope)
		if scopeType == "" {
			return fmt.Errorf("无效 scope 格式，应如 contact:<id>")
		}

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		m := &model.Memo{
			ID:        model.MemoID(),
			ScopeType: scopeType,
			ScopeID:   scopeID,
			Text:      text,
			Decay:     decay,
			ValidFrom: validFrom,
		}

		if err := s.Memos.Insert(m); err != nil {
			return err
		}

		if format == "json" {
			out, _ := json.Marshal(map[string]string{"id": m.ID})
			fmt.Println(string(out))
		} else {
			fmt.Printf("已写入记忆: %s\n", m.ID)
		}
		return nil
	},
}

var memoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出记忆",
	RunE: func(cmd *cobra.Command, args []string) error {
		scope, _ := cmd.Flags().GetString("scope")
		if scope == "" {
			return fmt.Errorf("--scope 是必需的")
		}

		scopeType, scopeID := parseScope(scope)

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		memos, err := s.Memos.ListByScope(scopeType, scopeID)
		if err != nil {
			return err
		}

		if format == "json" {
			out, _ := json.Marshal(memos)
			fmt.Println(string(out))
		} else {
			if len(memos) == 0 {
				fmt.Println("无记忆")
				return nil
			}
			for _, m := range memos {
				status := ""
				if m.Expired {
					status = " [过期]"
				}
				fmt.Printf("- %s%s\n", m.Text, status)
			}
		}
		return nil
	},
}

var memoryRecallCmd = &cobra.Command{
	Use:   "recall <query>",
	Short: "检索记忆",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		scope, _ := cmd.Flags().GetString("scope")
		topK, _ := cmd.Flags().GetInt("top-k")
		asOf, _ := cmd.Flags().GetString("as-of")

		if topK <= 0 {
			topK = 5
		}

		scopeType, scopeID := parseScope(scope)

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		engine := search.NewEngine(s.DB.DB)
		results, err := engine.SearchMemos(args[0], scopeType, scopeID, topK)
		if err != nil {
			memos, oldErr := s.Memos.Search(args[0], scopeType, scopeID, topK)
			if oldErr != nil {
				return err
			}
			var active []search.MemoResult
			for _, m := range memos {
				if !m.IsExpired(asOf) {
					active = append(active, search.MemoResult{
						ID: m.ID, ScopeType: m.ScopeType, ScopeID: m.ScopeID,
						Text: m.Text, Source: m.Source, Actor: m.Actor,
						CreatedAt: m.CreatedAt, Expired: m.Expired,
					})
				}
			}
			results = active
		} else {
			var active []search.MemoResult
			for _, m := range results {
				if !m.Expired {
					active = append(active, m)
				}
			}
			results = active
		}

		if format == "json" {
			out, _ := json.Marshal(results)
			fmt.Println(string(out))
		} else {
			if len(results) == 0 {
				fmt.Println("未找到匹配的记忆")
				return nil
			}
			for _, m := range results {
				fmt.Printf("- %s", m.Text)
				if m.Score > 0 {
					fmt.Printf(" [%.2f]", m.Score)
				}
				fmt.Println()
			}
		}
		return nil
	},
}

var memoryForgetCmd = &cobra.Command{
	Use:   "forget <memo-id>",
	Short: "删除记忆",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		if err := s.Memos.Delete(args[0]); err != nil {
			return err
		}
		fmt.Println("已删除记忆")
		return nil
	},
}

var memoryProposeCmd = &cobra.Command{
	Use:   "propose",
	Short: "提议新的记忆（带矛盾检测）",
	RunE: func(cmd *cobra.Command, args []string) error {
		scope, _ := cmd.Flags().GetString("scope")
		statement, _ := cmd.Flags().GetString("statement")
		sourceSnippet, _ := cmd.Flags().GetString("source-snippet")
		confidence, _ := cmd.Flags().GetFloat64("confidence")

		if scope == "" || statement == "" {
			return fmt.Errorf("--scope 和 --statement 是必需的")
		}

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		p, err := s.Memos.Propose(scope, statement, sourceSnippet, getActor(), confidence)
		if err != nil {
			return err
		}

		if err := s.FS.AppendProposal(p); err != nil {
			return err
		}

		if format == "json" {
			out, _ := json.Marshal(p)
			fmt.Println(string(out))
		} else {
			fmt.Printf("提议已记录: %s (status: %s)\n", p.ID, p.Status)
			if p.Status == "conflict" {
				fmt.Printf("  检测到 %d 条冲突:\n", len(p.ConflictWith))
				for _, c := range p.ConflictWith {
					fmt.Printf("    - %s\n", c.Text)
				}
				fmt.Printf("  建议动作: %s\n", p.SuggestedAction)
			}
		}
		return nil
	},
}

var memoryCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "提交/裁决提议",
	RunE: func(cmd *cobra.Command, args []string) error {
		proposalID, _ := cmd.Flags().GetString("proposal-id")
		action, _ := cmd.Flags().GetString("action")

		if proposalID == "" || action == "" {
			return fmt.Errorf("--proposal-id 和 --action 是必需的")
		}

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		if err := s.Memos.Commit(proposalID, action); err != nil {
			return err
		}

		fmt.Printf("已处理提议 %s (action: %s)\n", proposalID, action)
		return nil
	},
}

var memoryDecayScanCmd = &cobra.Command{
	Use:   "decay-scan",
	Short: "扫描并标记过期记忆",
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		count, err := s.Memos.DecayScan(dryRun)
		if err != nil {
			return err
		}

		if dryRun {
			fmt.Printf("发现 %d 条将要过期的记忆（预览模式，未实际标记）\n", count)
		} else {
			fmt.Printf("已标记 %d 条过期记忆\n", count)
		}
		return nil
	},
}

func parseScope(scope string) (scopeType, scopeID string) {
	parts := strings.SplitN(scope, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

func init() {
	memoryCmd.AddCommand(memoryWriteCmd)
	memoryCmd.AddCommand(memoryListCmd)
	memoryCmd.AddCommand(memoryRecallCmd)
	memoryCmd.AddCommand(memoryForgetCmd)
	memoryCmd.AddCommand(memoryProposeCmd)
	memoryCmd.AddCommand(memoryCommitCmd)
	memoryCmd.AddCommand(memoryDecayScanCmd)

	memoryWriteCmd.Flags().String("scope", "", "作用域 (contact:<id> 或 deal:<id>)")
	memoryWriteCmd.Flags().String("text", "", "记忆内容")
	memoryWriteCmd.Flags().String("decay", "180d", "衰减策略")
	memoryWriteCmd.Flags().String("valid-from", "", "生效时间")

	memoryListCmd.Flags().String("scope", "", "作用域")

	memoryRecallCmd.Flags().String("scope", "", "作用域")
	memoryRecallCmd.Flags().Int("top-k", 5, "返回数量上限")
	memoryRecallCmd.Flags().String("as-of", "", "查看指定时间点")

	memoryProposeCmd.Flags().String("scope", "", "作用域")
	memoryProposeCmd.Flags().String("statement", "", "陈述内容")
	memoryProposeCmd.Flags().String("source-snippet", "", "来源原文片段")
	memoryProposeCmd.Flags().Float64("confidence", 0.8, "置信度 (0-1)")

	memoryCommitCmd.Flags().String("proposal-id", "", "提议 ID")
	memoryCommitCmd.Flags().String("action", "", "动作: supersede|keep-both|reject")

	memoryDecayScanCmd.Flags().Bool("dry-run", false, "仅预览不执行")
}
