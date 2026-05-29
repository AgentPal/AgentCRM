package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/AgentPal/AgentCRM/internal/model"
	"github.com/spf13/cobra"
)

var dealCmd = &cobra.Command{
	Use:   "deal",
	Short: "管理商机",
}

var dealCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "创建商机",
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		contactID, _ := cmd.Flags().GetString("contact")
		amount, _ := cmd.Flags().GetInt("amount")
		currency, _ := cmd.Flags().GetString("currency")
		stage, _ := cmd.Flags().GetString("stage")

		if title == "" {
			return ErrDealTitleRequired
		}

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		d := model.NewDeal()
		d.ID = model.DealID()
		slug := model.Slugify(title)
		if slug == "" {
			slug = d.ID
		}
		d.Slug = slug
		if len(d.Slug) < 8 && d.Slug != d.ID {
			d.Slug = slug + "-" + d.ID[:8]
		}
		d.Title = title
		d.ContactIDs = []string{contactID}
		d.Amount = amount
		d.Currency = currency
		if stage != "" {
			d.Stage = stage
		}
		d.StageHistory = append(d.StageHistory, model.StageHistory{
			Stage:     d.Stage,
			EnteredAt: d.CreatedAt[:10],
			By:        getActor(),
		})

		if err := s.Deals.Create(d); err != nil {
			return err
		}
		if err := s.FS.WriteDeal(d); err != nil {
			return fmt.Errorf("write deal file: %w", err)
		}

		s.Events.Append(getActor(), model.EventDealCreated, map[string]string{
			"id": d.ID, "title": d.Title, "stage": d.Stage})

		if format == "json" {
			out, _ := json.Marshal(map[string]string{
				"id":    d.ID,
				"slug":  d.Slug,
				"title": d.Title,
			})
			fmt.Println(string(out))
		} else {
			fmt.Printf("已创建商机: %s (%s)\n", d.Title, d.ID)
			fmt.Printf("  金额: %d %s\n", d.Amount, d.Currency)
			fmt.Printf("  阶段: %s\n", d.Stage)
		}
		return nil
	},
}

var dealUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "更新商机字段（阶段/金额/其他）",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sets, _ := cmd.Flags().GetStringArray("set")
		reason, _ := cmd.Flags().GetString("reason")

		if len(sets) == 0 {
			return ErrDealSetRequired
		}

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		d, err := s.Deals.GetByID(args[0])
		if err != nil {
			return fmt.Errorf("deal not found: %w", err)
		}

		for _, set := range sets {
			parts := split2(set, "=")
			if parts == nil {
				return fmt.Errorf("invalid --set format: %s (expected field=value)", set)
			}
			field, value := parts[0], parts[1]

			switch field {
			case "stage":
				if !model.StageTransitionAllowed(d.Stage, value) {
					return fmt.Errorf("%w: cannot transition from %s to %s", ErrDealStageIrreversible, d.Stage, value)
				}
				if d.Stage == value {
					if format != "json" {
						fmt.Printf("阶段已经是 %s\n", value)
					}
					continue
				}
				oldStage := d.Stage

				if err := s.FS.UpdateDealField(d.Slug, "stage", value, oldStage, reason, getActor()); err != nil {
					return fmt.Errorf("update stage: %w", err)
				}
				s.Deals.UpdateField(args[0], "stage", value)

				eventType := model.EventDealStageChanged
				if value == "won" {
					eventType = model.EventDealClosedWon
				} else if value == "lost" {
					eventType = model.EventDealClosedLost
				}
				s.Events.Append(getActor(), eventType, map[string]string{
					"id": args[0], "from": oldStage, "to": value, "reason": reason})

				if format != "json" {
					fmt.Printf("阶段: %s → %s", oldStage, value)
					if reason != "" {
						fmt.Printf(" (%s)", reason)
					}
					fmt.Println()
				}

			case "amount":
				var newAmt int
				fmt.Sscanf(value, "%d", &newAmt)

				if d.Amount > 0 {
					pct := float64(newAmt-d.Amount) / float64(d.Amount) * 100
					if pct > 20 || pct < -20 {
						fmt.Printf("⚠ 金额变动超过 20%%（%.0f%%），请确认\n", pct)
					}
				}

				oldAmt := fmt.Sprintf("%d", d.Amount)
				if err := s.FS.UpdateDealField(d.Slug, "amount", value, oldAmt, reason, getActor()); err != nil {
					return fmt.Errorf("update amount: %w", err)
				}
				s.Deals.UpdateField(args[0], "amount", newAmt)
				s.Events.Append(getActor(), model.EventDealAmountChanged, map[string]string{
					"id": args[0], "from": oldAmt, "to": value, "reason": reason})

				if format != "json" {
					fmt.Printf("金额: %d → %s", d.Amount, value)
					if reason != "" {
						fmt.Printf(" (%s)", reason)
					}
					fmt.Println()
				}

			default:
				if err := s.Deals.UpdateField(args[0], field, value); err != nil {
					return err
				}
				if format != "json" {
					fmt.Printf("%s 已更新\n", field)
				}
			}
		}
		return nil
	},
}

var dealListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出商机",
	RunE: func(cmd *cobra.Command, args []string) error {
		stage, _ := cmd.Flags().GetString("stage")
		stageNotIn, _ := cmd.Flags().GetString("stage-not-in")
		owner, _ := cmd.Flags().GetString("owner")

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		results, err := s.Deals.List(stage, stageNotIn, owner)
		if err != nil {
			return err
		}

		if format == "json" {
			out, _ := json.Marshal(results)
			fmt.Println(string(out))
		} else {
			if len(results) == 0 {
				fmt.Println("无商机")
				return nil
			}
			for _, r := range results {
				fmt.Printf("%s [%s]", r.Title, r.Stage)
				if r.Amount > 0 {
					fmt.Printf(" %d%s", r.Amount, r.Currency)
				}
				fmt.Println()
			}
			fmt.Printf("\n共 %d 个商机\n", len(results))
		}
		return nil
	},
}

var dealGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "获取商机详情",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		asOf, _ := cmd.Flags().GetString("as-of")

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		dIndex, err := s.Deals.GetByID(args[0])
		if err != nil {
			return err
		}
		d, err := s.FS.ReadDeal(dIndex.Slug)
		if err != nil {
			d = dIndex
		}

		if asOf != "" {
			d = d.AsOf(asOf)
		}

		if format == "json" {
			out, _ := json.Marshal(d)
			fmt.Println(string(out))
		} else {
			if asOf != "" {
				fmt.Printf("[时间点: %s]\n", asOf)
			}
			fmt.Printf("ID: %s\n", d.ID)
			fmt.Printf("名称: %s\n", d.Title)
			fmt.Printf("阶段: %s\n", d.Stage)
			fmt.Printf("金额: %d %s\n", d.Amount, d.Currency)
			if d.ExpectedCloseAt != "" {
				fmt.Printf("预计成交: %s\n", d.ExpectedCloseAt[:10])
			}
			if len(d.ContactIDs) > 0 {
				fmt.Printf("联系人: %s\n", strings.Join(d.ContactIDs, ", "))
			}
			if len(d.StageHistory) > 0 {
				fmt.Printf("阶段历程:\n")
				for _, h := range d.StageHistory {
					fmt.Printf("  %s (%s)", h.Stage, h.EnteredAt)
					if h.By != "" {
						fmt.Printf(", by: %s", h.By)
					}
					fmt.Println()
				}
			}
		}
		return nil
	},
}

var dealHistoryCmd = &cobra.Command{
	Use:   "history <id>",
	Short: "查看商机字段历史",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		field, _ := cmd.Flags().GetString("field")
		if field == "" {
			return ErrDealFieldRequired
		}

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		dIndex, err := s.Deals.GetByID(args[0])
		if err != nil {
			return fmt.Errorf("deal not found: %w", err)
		}
		d, err := s.FS.ReadDeal(dIndex.Slug)
		if err != nil {
			return fmt.Errorf("read deal file: %w", err)
		}

		switch field {
		case "stage":
			if len(d.StageHistory) == 0 {
				fmt.Println("无阶段历史")
				return nil
			}
			if format == "json" {
				out, _ := json.Marshal(d.StageHistory)
				fmt.Println(string(out))
				return nil
			}
			for _, h := range d.StageHistory {
				fmt.Printf("stage: %s (entered: %s)", h.Stage, h.EnteredAt)
				if h.By != "" {
					fmt.Printf(", by: %s", h.By)
				}
				if h.Reason != "" {
					fmt.Printf(", reason: %s", h.Reason)
				}
				fmt.Println()
			}
		case "amount":
			if len(d.AmountHistory) == 0 {
				fmt.Println("无金额历史")
				return nil
			}
			if format == "json" {
				out, _ := json.Marshal(d.AmountHistory)
				fmt.Println(string(out))
				return nil
			}
			for _, h := range d.AmountHistory {
				to := h.To
				if to == "~" || to == "" {
					to = "至今"
				}
				fmt.Printf("amount: %d (from: %s, to: %s)", h.Value, h.From, to)
				if h.Reason != "" {
					fmt.Printf(", reason: %s", h.Reason)
				}
				fmt.Println()
			}
		default:
			return fmt.Errorf("%w: unsupported field: %s (supported: stage, amount)", ErrDealFieldUnsupported, field)
		}
		return nil
	},
}

var dealSummarizeCmd = &cobra.Command{
	Use:   "summarize <id>",
	Short: "生成商机自然语言总结",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		d, err := s.Deals.GetByID(args[0])
		if err != nil {
			return err
		}

		if format == "json" {
			out, _ := json.Marshal(d)
			fmt.Println(string(out))
			return nil
		}

		df, err := s.FS.ReadDeal(d.Slug)
		if err != nil {
			df = d
		}

		created, _ := time.Parse(time.RFC3339, d.CreatedAt)
		daysSinceCreated := int(time.Since(created).Hours() / 24)
		if daysSinceCreated < 0 {
			daysSinceCreated = 0
		}

		fmt.Printf("商机: %s\n", d.Title)
		fmt.Printf("当前阶段: %s", d.Stage)
		if len(df.StageHistory) > 0 {
			lastStage := df.StageHistory[len(df.StageHistory)-1]
			entered, _ := time.Parse("2006-01-02", lastStage.EnteredAt)
			daysInStage := int(time.Since(entered).Hours() / 24)
			if daysInStage < 0 {
				daysInStage = 0
			}
			fmt.Printf(" (已停留 %d 天)", daysInStage)
		}
		fmt.Println()
		fmt.Printf("金额: %d %s\n", d.Amount, d.Currency)
		fmt.Printf("创建: %d 天前\n", daysSinceCreated)
		if d.ExpectedCloseAt != "" {
			fmt.Printf("预计成交: %s\n", d.ExpectedCloseAt[:10])
		}
		if d.Owner != "" {
			fmt.Printf("负责人: %s\n", d.Owner)
		}
		if len(df.StageHistory) > 1 {
			history := make([]string, len(df.StageHistory))
			for i, h := range df.StageHistory {
				history[i] = h.Stage
			}
			fmt.Printf("阶段历程: %s\n", strings.Join(history, " → "))
		}
		return nil
	},
}

func init() {
	dealCmd.AddCommand(dealCreateCmd)
	dealCmd.AddCommand(dealUpdateCmd)
	dealCmd.AddCommand(dealListCmd)
	dealCmd.AddCommand(dealGetCmd)
	dealCmd.AddCommand(dealHistoryCmd)
	dealCmd.AddCommand(dealSummarizeCmd)

	dealCreateCmd.Flags().String("title", "", "商机名称（必需）")
	dealCreateCmd.Flags().String("contact", "", "关联联系人 ID")
	dealCreateCmd.Flags().Int("amount", 0, "金额")
	dealCreateCmd.Flags().String("currency", "CNY", "币种")
	dealCreateCmd.Flags().String("stage", "lead", "起始阶段")

	dealUpdateCmd.Flags().StringArray("set", nil, "设置字段 (field=value)")
	dealUpdateCmd.Flags().String("reason", "", "变更原因")

	dealListCmd.Flags().String("stage", "", "按阶段过滤")
	dealListCmd.Flags().String("stage-not-in", "", "排除的阶段（逗号分隔）")
	dealListCmd.Flags().String("owner", "", "按负责人过滤")

	dealGetCmd.Flags().String("as-of", "", "查看指定时间点的数据")

	dealHistoryCmd.Flags().String("field", "", "字段名")
}

func split2(s, sep string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return nil
}
