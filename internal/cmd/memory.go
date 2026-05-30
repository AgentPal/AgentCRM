package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AgentPal/AgentCRM/internal/i18n"
	"github.com/AgentPal/AgentCRM/internal/model"
	"github.com/AgentPal/AgentCRM/internal/search"
	"github.com/spf13/cobra"
)

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: i18n.T("cmd.memory.short"),
}

var memoryWriteCmd = &cobra.Command{
	Use:   "write",
	Short: i18n.T("cmd.memory.write.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		scope, _ := cmd.Flags().GetString("scope")
		text, _ := cmd.Flags().GetString("text")
		decay, _ := cmd.Flags().GetString("decay")
		validFrom, _ := cmd.Flags().GetString("valid-from")

		if scope == "" || text == "" {
			return ErrMemoryScopeTextRequired
		}

		scopeType, scopeID := parseScope(scope)
		if scopeType == "" {
			// format string is a translated message; placeholder consistency enforced by TestPlaceholderConsistency
		return fmt.Errorf(i18n.T("error.memory.scope.format"))
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
			fmt.Println(i18n.T("output.memory.written", m.ID))
		}
		return nil
	},
}

var memoryListCmd = &cobra.Command{
	Use:   "list",
	Short: i18n.T("cmd.memory.list.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		scope, _ := cmd.Flags().GetString("scope")
		if scope == "" {
			return ErrMemoryScopeRequired
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
				fmt.Println(i18n.T("output.memory.list.none"))
				return nil
			}
			for _, m := range memos {
				status := ""
				if m.Expired {
					status = i18n.T("output.memory.list.expired")
				}
				fmt.Printf("- %s%s\n", m.Text, status)
			}
		}
		return nil
	},
}

var memoryRecallCmd = &cobra.Command{
	Use:   "recall <query>",
	Short: i18n.T("cmd.memory.recall.short"),
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
				fmt.Println(i18n.T("output.memory.recall.none"))
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
	Short: i18n.T("cmd.memory.forget.short"),
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
		fmt.Println(i18n.T("output.memory.forget.ok"))
		return nil
	},
}

var memoryProposeCmd = &cobra.Command{
	Use:   "propose",
	Short: i18n.T("cmd.memory.propose.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		scope, _ := cmd.Flags().GetString("scope")
		statement, _ := cmd.Flags().GetString("statement")
		sourceSnippet, _ := cmd.Flags().GetString("source-snippet")
		confidence, _ := cmd.Flags().GetFloat64("confidence")

		if scope == "" || statement == "" {
			return ErrMemoryScopeStatementRequired
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
			fmt.Println(i18n.T("output.memory.propose.recorded", p.ID, p.Status))
			if p.Status == "conflict" {
				fmt.Println(i18n.T("output.memory.propose.conflicts", len(p.ConflictWith)))
				for _, c := range p.ConflictWith {
					fmt.Printf("    - %s\n", c.Text)
				}
				fmt.Println(i18n.T("output.memory.propose.suggested_action", p.SuggestedAction))
			}
		}
		return nil
	},
}

var memoryCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: i18n.T("cmd.memory.commit.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		proposalID, _ := cmd.Flags().GetString("proposal-id")
		action, _ := cmd.Flags().GetString("action")

		if proposalID == "" || action == "" {
			return ErrMemoryProposalActionRequired
		}

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		if err := s.Memos.Commit(proposalID, action); err != nil {
			return err
		}

		fmt.Println(i18n.T("output.memory.commit.done", proposalID, action))
		return nil
	},
}

var memoryDecayScanCmd = &cobra.Command{
	Use:   "decay-scan",
	Short: i18n.T("cmd.memory.decay.short"),
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
			fmt.Println(i18n.T("output.memory.decay.preview", count))
		} else {
			fmt.Println(i18n.T("output.memory.decay.done", count))
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

	memoryWriteCmd.Flags().String("scope", "", i18n.T("flag.memory.scope"))
	memoryWriteCmd.Flags().String("text", "", i18n.T("flag.memory.text"))
	memoryWriteCmd.Flags().String("decay", "180d", i18n.T("flag.memory.decay"))
	memoryWriteCmd.Flags().String("valid-from", "", i18n.T("flag.memory.valid_from"))

	memoryListCmd.Flags().String("scope", "", i18n.T("flag.memory.scope"))

	memoryRecallCmd.Flags().String("scope", "", i18n.T("flag.memory.scope"))
	memoryRecallCmd.Flags().Int("top-k", 5, i18n.T("flag.memory.top_k"))
	memoryRecallCmd.Flags().String("as-of", "", i18n.T("flag.memory.as_of"))

	memoryProposeCmd.Flags().String("scope", "", i18n.T("flag.memory.scope"))
	memoryProposeCmd.Flags().String("statement", "", i18n.T("flag.memory.statement"))
	memoryProposeCmd.Flags().String("source-snippet", "", i18n.T("flag.memory.source_snippet"))
	memoryProposeCmd.Flags().Float64("confidence", 0.8, i18n.T("flag.memory.confidence"))

	memoryCommitCmd.Flags().String("proposal-id", "", i18n.T("flag.memory.proposal_id"))
	memoryCommitCmd.Flags().String("action", "", i18n.T("flag.memory.action"))

	memoryDecayScanCmd.Flags().Bool("dry-run", false, i18n.T("flag.memory.dry_run"))
}
