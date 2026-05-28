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

var contactCmd = &cobra.Command{
	Use:   "contact",
	Short: i18n.T("cmd.contact.short"),
}

var contactUpsertCmd = &cobra.Command{
	Use:   "upsert",
	Short: i18n.T("cmd.contact.upsert.short"),
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
			if created {
				fmt.Printf(i18n.T("output.contact.upsert.created")+"\n", c.Name, c.ID)
			} else {
				fmt.Printf(i18n.T("output.contact.upsert.updated")+"\n", c.Name, c.ID)
			}
			if c.Company != "" {
				fmt.Println(i18n.T("output.contact.upsert.company", c.Company))
			}
			if c.Title != "" {
				fmt.Println(i18n.T("output.contact.upsert.title", c.Title))
			}
		}
		return nil
	},
}

var contactGetCmd = &cobra.Command{
	Use:   "get <id-or-slug>",
	Short: i18n.T("cmd.contact.get.short"),
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
				fmt.Println(i18n.T("output.contact.get.as_of", asOf))
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
	Short: i18n.T("cmd.contact.search.short"),
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
				fmt.Println(i18n.T("output.contact.search.none"))
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
					fmt.Printf(i18n.T("output.contact.search.recent"), r.LastActivityAt[:10])
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
	Short: i18n.T("cmd.contact.list.short"),
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
				fmt.Println(i18n.T("output.contact.list.none"))
				return nil
			}
			for _, r := range results {
				fmt.Printf("%s", r.Name)
				if r.Company != "" {
					fmt.Printf(" - %s", r.Company)
				}
				fmt.Println()
			}
			fmt.Printf(i18n.T("output.contact.list.count"), len(results))
		}
		return nil
	},
}

var contactUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: i18n.T("cmd.contact.update.short"),
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
					fmt.Println(i18n.T("output.contact.update.same", field))
				} else if oldVal != "" {
					fmt.Println(i18n.T("output.contact.update.changed", field, oldVal, value))
				} else {
					fmt.Println(i18n.T("output.contact.update.set", field, value))
				}
			}
		}
		return nil
	},
}

var contactHistoryCmd = &cobra.Command{
	Use:   "history <id>",
	Short: i18n.T("cmd.contact.history.short"),
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
			fmt.Println(i18n.T("output.contact.history.none"))
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
				to = i18n.T("output.contact.history.to")
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
	Short: i18n.T("cmd.contact.merge.short"),
	Long:  i18n.T("cmd.contact.merge.long"),
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
			fmt.Println(i18n.T("output.contact.merge.success", keeper.Name, keeper.ID, mergee.Name, mergee.ID))
			if reason != "" {
				fmt.Println(i18n.T("output.contact.merge.reason", reason))
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

	contactUpsertCmd.Flags().String("name", "", i18n.T("flag.contact.name"))
	contactUpsertCmd.Flags().String("email", "", i18n.T("flag.contact.email"))
	contactUpsertCmd.Flags().String("company", "", i18n.T("flag.contact.company"))
	contactUpsertCmd.Flags().String("title", "", i18n.T("flag.contact.title"))
	contactUpsertCmd.Flags().String("phone", "", i18n.T("flag.contact.phone"))
	contactUpsertCmd.Flags().String("source", "", i18n.T("flag.contact.source"))
	contactUpsertCmd.Flags().StringSlice("tag", nil, i18n.T("flag.contact.tag"))

	contactGetCmd.Flags().String("as-of", "", i18n.T("flag.contact.as_of"))

	contactSearchCmd.Flags().Int("limit", 10, i18n.T("flag.contact.limit"))

	contactListCmd.Flags().String("tag", "", i18n.T("flag.contact.tag_filter"))
	contactListCmd.Flags().String("updated-since", "", i18n.T("flag.contact.updated_since"))
	contactListCmd.Flags().String("has-field", "", i18n.T("flag.contact.has_field"))
	contactListCmd.Flags().Int("limit", 50, i18n.T("flag.contact.limit"))

	contactUpdateCmd.Flags().StringArray("set", nil, i18n.T("flag.contact.set"))
	contactUpdateCmd.Flags().String("reason", "", i18n.T("flag.contact.reason"))

	contactHistoryCmd.Flags().String("field", "", i18n.T("flag.contact.field"))

	contactMergeCmd.Flags().String("reason", "", i18n.T("flag.contact.merge_reason"))
}
