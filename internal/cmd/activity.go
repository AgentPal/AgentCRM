package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/AgentPal/AgentCRM/internal/model"
	"github.com/spf13/cobra"
)

var activityCmd = &cobra.Command{
	Use:   "activity",
	Short: "管理活动记录",
}

var activityLogCmd = &cobra.Command{
	Use:   "log",
	Short: "记录一次互动",
	RunE: func(cmd *cobra.Command, args []string) error {
		contactID, _ := cmd.Flags().GetString("contact")
		dealID, _ := cmd.Flags().GetString("deal")
		actType, _ := cmd.Flags().GetString("type")
		direction, _ := cmd.Flags().GetString("direction")
		channel, _ := cmd.Flags().GetString("channel")
		summary, _ := cmd.Flags().GetString("summary")
		dedupeKey, _ := cmd.Flags().GetString("dedupe-key")
		bodyFile, _ := cmd.Flags().GetString("body-file")

		if contactID == "" {
			return ErrActivityContactRequired
		}
		if summary == "" {
			return ErrActivitySummaryRequired
		}
		if dedupeKey == "" {
			return ErrActivityDedupeKeyRequired
		}

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		a := model.NewActivity()
		a.ID = model.ActivityID()
		a.ContactIDs = []string{contactID}
		a.Type = actType
		a.Direction = direction
		a.Channel = channel
		a.Summary = summary
		a.DedupeKey = dedupeKey
		a.Actor = getActor()

		if dealID != "" {
			a.DealIDs = []string{dealID}
		}
		if bodyFile != "" {
			body, err := os.ReadFile(bodyFile)
			if err != nil {
				return fmt.Errorf("read body file: %w", err)
			}
			a.Body = string(body)
		}

		created, err := s.Activities.Log(a)
		if err != nil {
			return err
		}

		if err := s.FS.AppendActivity(a); err != nil {
			return fmt.Errorf("write activity file: %w", err)
		}

		// 发送事件
		s.Events.Append(getActor(), model.EventActivityLogged, map[string]string{
			"id": a.ID, "contact": contactID, "type": actType, "summary": summary})

		if format == "json" {
			out, _ := json.Marshal(map[string]interface{}{
				"id":      a.ID,
				"created": created,
				"ts":      a.Timestamp,
			})
			fmt.Println(string(out))
		} else {
			if !created {
				fmt.Printf("活动已存在（重复）: %s\n", a.ID)
			} else {
				fmt.Printf("已记录活动: %s\n", a.ID)
				fmt.Printf("  类型: %s\n", a.Type)
				fmt.Printf("  摘要: %s\n", a.Summary)
			}
		}
		return nil
	},
}

var activityListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出活动",
	RunE: func(cmd *cobra.Command, args []string) error {
		contactID, _ := cmd.Flags().GetString("contact")
		since, _ := cmd.Flags().GetString("since")
		actType, _ := cmd.Flags().GetString("type")
		searchQuery, _ := cmd.Flags().GetString("search")
		limit, _ := cmd.Flags().GetInt("limit")

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		var activities []model.Activity
		if searchQuery != "" {
			activities, err = s.Activities.Search(searchQuery, limit)
		} else {
			activities, err = s.Activities.ListByContact(contactID, since, actType, limit)
		}
		if err != nil {
			return err
		}

		if format == "json" {
			out, _ := json.Marshal(activities)
			fmt.Println(string(out))
		} else {
			if len(activities) == 0 {
				fmt.Println("无活动记录")
				return nil
			}
			for _, a := range activities {
				ts := a.Timestamp
				if len(ts) > 16 {
					ts = ts[:16]
				}
				fmt.Printf("[%s] %s", ts, a.Type)
				if a.Direction != "" {
					fmt.Printf(" (%s)", a.Direction)
				}
				fmt.Printf(": %s\n", a.Summary)
			}
			fmt.Printf("\n共 %d 条记录\n", len(activities))
		}
		return nil
	},
}

var timelineCmd = &cobra.Command{
	Use:   "timeline <contact-id>",
	Short: "查看联系人的时间线（含活动 + 商机变动）",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		since, _ := cmd.Flags().GetString("since")
		detail, _ := cmd.Flags().GetString("detail")
		limit := 50
		if detail == "full" {
			limit = 200
		}

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		// 获取联系人的活动
		activities, err := s.Activities.ListByContact(args[0], since, "", limit)
		if err != nil {
			return err
		}

		// 获取关联商机变动
		contact, err := s.Contacts.GetByID(args[0])
		var dealChanges []string
		if err == nil {
			deals, _ := s.Deals.List("", "", "")
			for _, d := range deals {
				isRelated := false

				if d.ContactName != "" && strings.Contains(d.ContactName, contact.Name) {
					isRelated = true
				} else {
					deal, err := s.Deals.GetByID(d.ID)
					if err != nil {
						continue
					}
					for _, cid := range deal.ContactIDs {
						if cid == args[0] {
							isRelated = true
							break
						}
					}
				}

				if isRelated && d.UpdatedAt != "" && (since == "" || d.UpdatedAt >= since) {
					dealChanges = append(dealChanges, fmt.Sprintf("Deal: %s → %s (%d %s)", d.Title, d.Stage, d.Amount, d.Currency))
				}
			}
		}

		if format == "json" {
			out, _ := json.Marshal(map[string]interface{}{
				"activities":   activities,
				"deal_changes": dealChanges,
			})
			fmt.Println(string(out))
		} else {
			if len(activities) == 0 && len(dealChanges) == 0 {
				fmt.Println("无活动记录")
				return nil
			}

			// 显示活动
			for _, a := range activities {
				ts := a.Timestamp
				if len(ts) > 10 {
					ts = ts[:10]
				}
				fmt.Printf("[%s] %s", ts, a.Type)
				if a.Direction != "" {
					fmt.Printf(" (%s)", a.Direction)
				}
				if detail != "brief" {
					fmt.Printf(" | %s", a.Summary)
				}
				fmt.Println()
			}

			// 显示商机变动
			if len(dealChanges) > 0 {
				fmt.Println("\n--- 商机变动 ---")
				for _, c := range dealChanges {
					fmt.Println(c)
				}
			}

			fmt.Printf("\n共 %d 条活动", len(activities))
			if len(dealChanges) > 0 {
				fmt.Printf(", %d 条商机变动", len(dealChanges))
			}
			fmt.Println()
		}
		return nil
	},
}

func init() {
	activityCmd.AddCommand(activityLogCmd)
	activityCmd.AddCommand(activityListCmd)
	activityCmd.AddCommand(timelineCmd)

	activityLogCmd.Flags().String("contact", "", "联系人 ID（必需）")
	activityLogCmd.Flags().String("deal", "", "商机 ID")
	activityLogCmd.Flags().String("type", "note", "类型: email|call|meeting|chat|note|social")
	activityLogCmd.Flags().String("direction", "in", "方向: in|out")
	activityLogCmd.Flags().String("channel", "", "渠道: gmail|twitter|wechat|phone|...")
	activityLogCmd.Flags().String("summary", "", "一句话摘要（必需）")
	activityLogCmd.Flags().String("dedupe-key", "", "去重键（必需）")
	activityLogCmd.Flags().String("body-file", "", "正文文件路径")

	activityListCmd.Flags().String("contact", "", "联系人 ID")
	activityListCmd.Flags().String("since", "", "起始时间")
	activityListCmd.Flags().String("type", "", "活动类型")
	activityListCmd.Flags().String("search", "", "搜索摘要和正文")
	activityListCmd.Flags().Int("limit", 50, "返回数量上限")

	timelineCmd.Flags().String("since", "", "起始时间")
	timelineCmd.Flags().String("detail", "standard", "详细程度: brief|standard|full")
}
