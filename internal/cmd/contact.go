package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AgentPal/AgentCRM/internal/model"
	"github.com/AgentPal/AgentCRM/internal/search"
	"github.com/spf13/cobra"
)

var contactCmd = &cobra.Command{
	Use:   "contact",
	Short: "管理联系人",
}

var contactUpsertCmd = &cobra.Command{
	Use:   "upsert",
	Short: "创建或更新联系人（按 email 去重）",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		email, _ := cmd.Flags().GetString("email")
		company, _ := cmd.Flags().GetString("company")
		title, _ := cmd.Flags().GetString("title")
		phone, _ := cmd.Flags().GetString("phone")
		source, _ := cmd.Flags().GetString("source")
		tags, _ := cmd.Flags().GetStringSlice("tag")

		if name == "" {
			return ErrContactNameRequired
		}

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		c := model.NewContact()
		c.ID = model.ContactID()
		slug := model.Slugify(name)
		if slug == "" {
			slug = c.ID
		}
		c.Slug = slug
		if len(c.Slug) < 8 && c.Slug != c.ID {
			c.Slug = slug + "-" + c.ID[:8]
		}
		c.Name = name
		c.Emails = []string{email}
		c.Company = company
		c.Title = title
		c.Source = source
		c.Tags = tags
		if phone != "" {
			c.Phones = []string{phone}
		}

		created, err := s.Contacts.Upsert(c)
		if err != nil {
			return err
		}

		if err := s.FS.WriteContact(c); err != nil {
			return fmt.Errorf("write contact file: %w", err)
		}

		eventType := model.EventContactCreated
		if !created {
			eventType = model.EventContactUpdated
		}
		s.Events.Append(getActor(), eventType, map[string]string{"id": c.ID, "name": c.Name})

		if format == "json" {
			out, _ := json.Marshal(map[string]interface{}{
				"id":      c.ID,
				"slug":    c.Slug,
				"created": created,
			})
			fmt.Println(string(out))
		} else {
			action := "已更新"
			if created {
				action = "已创建"
			}
			fmt.Printf("%s 联系人: %s (%s)\n", action, c.Name, c.ID)
			if c.Company != "" {
				fmt.Printf("  公司: %s\n", c.Company)
			}
			if c.Title != "" {
				fmt.Printf("  职位: %s\n", c.Title)
			}
		}
		return nil
	},
}

var contactGetCmd = &cobra.Command{
	Use:   "get <id-or-slug>",
	Short: "获取联系人详情",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		asOf, _ := cmd.Flags().GetString("as-of")

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		c, err := s.Contacts.GetByID(args[0])
		if err != nil {
			c, err = s.FS.ReadContact(args[0])
			if err != nil {
				return fmt.Errorf("contact not found: %s", args[0])
			}
		}

		if asOf != "" {
			c = c.AsOf(asOf)
		}

		if format == "json" {
			out, _ := json.Marshal(c)
			fmt.Println(string(out))
		} else {
			if asOf != "" {
				fmt.Printf("[时间点: %s]\n", asOf)
			}
			fmt.Printf("ID: %s\n", c.ID)
			fmt.Printf("Name: %s\n", c.Name)
			if c.Company != "" {
				fmt.Printf("Company: %s\n", c.Company)
			}
			if c.Title != "" {
				fmt.Printf("Title: %s\n", c.Title)
			}
			if len(c.Emails) > 0 {
				fmt.Printf("Emails: %s\n", strings.Join(c.Emails, ", "))
			}
			if c.Birthday != "" {
				fmt.Printf("Birthday: %s\n", c.Birthday)
			}
			if len(c.Tags) > 0 {
				fmt.Printf("Tags: %s\n", strings.Join(c.Tags, ", "))
			}
		}
		return nil
	},
}

var contactSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "搜索联系人",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		if limit <= 0 {
			limit = 10
		}

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		engine := search.NewEngine(s.DB.DB)
		results, err := engine.SearchContacts(args[0], limit)
		if err != nil {
			oldResults, oldErr := s.Contacts.Search(args[0], limit)
			if oldErr != nil {
				return err
			}
			results = make([]search.ContactResult, len(oldResults))
			for i, r := range oldResults {
				results[i] = search.ContactResult{
					ID: r.ID, Name: r.Name, Company: r.Company,
					Title: r.Title, Email: r.Email,
					LastActivityAt: r.LastActivityAt, Tags: r.Tags,
				}
			}
		}

		if format == "json" {
			out, _ := json.Marshal(results)
			fmt.Println(string(out))
		} else {
			if len(results) == 0 {
				fmt.Println("未找到匹配的联系人")
				return nil
			}
			for i, r := range results {
				fmt.Printf("%d. %s", i+1, r.Name)
				if r.Company != "" {
					fmt.Printf(" (%s", r.Company)
					if r.Title != "" {
						fmt.Printf(", %s", r.Title)
					}
					fmt.Print(")")
				}
				if r.Email != "" {
					fmt.Printf(" - %s", r.Email)
				}
				if r.LastActivityAt != "" {
					fmt.Printf(" [最近: %s]", r.LastActivityAt[:10])
				}
				if r.Score > 0 {
					fmt.Printf(" [%.2f]", r.Score)
				}
				fmt.Println()
			}
		}
		return nil
	},
}

var contactListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出联系人",
	RunE: func(cmd *cobra.Command, args []string) error {
		tag, _ := cmd.Flags().GetString("tag")
		updatedSince, _ := cmd.Flags().GetString("updated-since")
		hasField, _ := cmd.Flags().GetString("has-field")
		limit, _ := cmd.Flags().GetInt("limit")

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		results, err := s.Contacts.List(tag, updatedSince, hasField, limit)
		if err != nil {
			return err
		}

		if format == "json" {
			out, _ := json.Marshal(results)
			fmt.Println(string(out))
		} else {
			if len(results) == 0 {
				fmt.Println("无联系人")
				return nil
			}
			for _, r := range results {
				fmt.Printf("%s", r.Name)
				if r.Company != "" {
					fmt.Printf(" - %s", r.Company)
				}
				fmt.Println()
			}
			fmt.Printf("\n共 %d 个联系人\n", len(results))
		}
		return nil
	},
}

var contactUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "更新联系人字段（自动归档旧值到 _history）",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sets, _ := cmd.Flags().GetStringArray("set")
		reason, _ := cmd.Flags().GetString("reason")

		if len(sets) == 0 {
			return ErrContactSetRequired
		}

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		c, err := s.Contacts.GetByID(args[0])
		if err != nil {
			return fmt.Errorf("contact not found: %w", err)
		}

		actor := getActor()

		for _, set := range sets {
			parts := split2(set, "=")
			if parts == nil {
				return fmt.Errorf("invalid --set format: %s (expected field=value)", set)
			}
			field, value := parts[0], parts[1]

			oldVal, err := s.FS.UpdateContactField(c.Slug, field, value, reason, actor)
			if err != nil {
				return fmt.Errorf("update %s: %w", field, err)
			}

			s.Contacts.Update(args[0], field, value, reason)

			if format != "json" {
				if oldVal == value {
					fmt.Printf("%s 已经是该值\n", field)
				} else if oldVal != "" {
					fmt.Printf("已更新 %s: %s → %s\n", field, oldVal, value)
				} else {
					fmt.Printf("已设置 %s = %s\n", field, value)
				}
			}
		}
		return nil
	},
}

var contactHistoryCmd = &cobra.Command{
	Use:   "history <id>",
	Short: "查看联系人字段历史",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		field, _ := cmd.Flags().GetString("field")
		if field == "" {
			return ErrContactFieldRequired
		}

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		cIndex, err := s.Contacts.GetByID(args[0])
		if err != nil {
			return fmt.Errorf("contact not found: %w", err)
		}
		c, err := s.FS.ReadContact(cIndex.Slug)
		if err != nil {
			return fmt.Errorf("read contact file: %w", err)
		}

		var history []model.FieldHistory
		switch field {
		case "company":
			history = c.CompanyHistory
		case "title":
			history = c.TitleHistory
		default:
			return fmt.Errorf("unsupported field: %s (supported: company, title)", field)
		}

		if len(history) == 0 {
			fmt.Println("无历史记录")
			return nil
		}

		if format == "json" {
			out, _ := json.Marshal(history)
			fmt.Println(string(out))
			return nil
		}

		for _, h := range history {
			to := h.To
			if to == "~" || to == "" {
				to = "至今"
			}
			fmt.Printf("%s: %s (from: %s, to: %s)", field, h.Value, h.From, to)
			if h.SetBy != "" {
				fmt.Printf(", by: %s", h.SetBy)
			}
			fmt.Println()
		}
		return nil
	},
}

var contactMergeCmd = &cobra.Command{
	Use:   "merge <keeper-id> <merge-id>",
	Short: "合并两个联系人",
	Long:  `将 merge-id 合并到 keeper-id。merge-id 的文件会被重命名为 .merged-into-<keeper-id>.md。`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		reason, _ := cmd.Flags().GetString("reason")

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		keeper, err := s.Contacts.GetByID(args[0])
		if err != nil {
			return fmt.Errorf("keeper not found: %w", err)
		}
		mergee, err := s.Contacts.GetByID(args[1])
		if err != nil {
			return fmt.Errorf("mergee not found: %w", err)
		}

		actor := getActor()

		if err := s.FS.MergeContactFiles(keeper.ID, mergee.ID, keeper.Slug, mergee.Slug, reason, actor); err != nil {
			return fmt.Errorf("merge files: %w", err)
		}

		s.Contacts.Delete(mergee.ID)

		if format != "json" {
			fmt.Printf("已合并: %s (%s) ← %s (%s)\n", keeper.Name, keeper.ID, mergee.Name, mergee.ID)
			if reason != "" {
				fmt.Printf("原因: %s\n", reason)
			}
		}
		return nil
	},
}

func init() {
	contactCmd.AddCommand(contactUpsertCmd)
	contactCmd.AddCommand(contactGetCmd)
	contactCmd.AddCommand(contactSearchCmd)
	contactCmd.AddCommand(contactListCmd)
	contactCmd.AddCommand(contactUpdateCmd)
	contactCmd.AddCommand(contactHistoryCmd)
	contactCmd.AddCommand(contactMergeCmd)

	contactUpsertCmd.Flags().String("name", "", "联系人姓名（必需）")
	contactUpsertCmd.Flags().String("email", "", "邮箱")
	contactUpsertCmd.Flags().String("company", "", "公司")
	contactUpsertCmd.Flags().String("title", "", "职位")
	contactUpsertCmd.Flags().String("phone", "", "电话")
	contactUpsertCmd.Flags().String("source", "", "来源")
	contactUpsertCmd.Flags().StringSlice("tag", nil, "标签")

	contactGetCmd.Flags().String("as-of", "", "查看指定时间点的数据")

	contactSearchCmd.Flags().Int("limit", 10, "返回数量上限")

	contactListCmd.Flags().String("tag", "", "按标签过滤")
	contactListCmd.Flags().String("updated-since", "", "按更新日期过滤")
	contactListCmd.Flags().String("has-field", "", "按存在字段过滤 (birthday)")
	contactListCmd.Flags().Int("limit", 50, "返回数量上限")

	contactUpdateCmd.Flags().StringArray("set", nil, "设置字段 (field=value)")
	contactUpdateCmd.Flags().String("reason", "", "变更原因")

	contactHistoryCmd.Flags().String("field", "", "字段名")

	contactMergeCmd.Flags().String("reason", "", "合并原因")
}
