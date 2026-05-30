package cmd

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AgentPal/AgentCRM/internal/i18n"
		"github.com/AgentPal/AgentCRM/internal/model"
	"github.com/AgentPal/AgentCRM/internal/store"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// exportCmd 导出数据。
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: i18n.T("cmd.io.export.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		formatFlag, _ := cmd.Flags().GetString("format")
		outDir, _ := cmd.Flags().GetString("out")

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		if err := os.MkdirAll(outDir, 0755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}

		switch formatFlag {
		case "json":
			return exportJSON(s, outDir)
		case "csv":
			return exportCSV(s, outDir)
		default:
			return fmt.Errorf("unsupported format: %s (supported: json, csv)", formatFlag)
		}
	},
}

// importCmd 导入数据。
var importCmd = &cobra.Command{
	Use:   "import",
	Short: i18n.T("cmd.io.import.short"),
	Long: `从外部源导入联系人数据。

来源:
  vcard  导入 vCard (.vcf) 文件
  csv    导入 CSV 文件（列: name,email,company,title,phone,tags,source）`,
	RunE: func(cmd *cobra.Command, args []string) error {
		source, _ := cmd.Flags().GetString("source")
		file, _ := cmd.Flags().GetString("file")

		if file == "" {
			// format string is a translated message; placeholder consistency enforced by TestPlaceholderConsistency
			return fmt.Errorf(i18n.T("error.io.file.required"))
		}

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		switch source {
		case "vcard":
			return importVCard(s, file)
		case "csv":
			return importCSV(s, file)
		default:
			return fmt.Errorf("unsupported source: %s (supported: vcard, csv)", source)
		}
	},
}

func init() {
	exportCmd.Flags().String("format", "json", i18n.T("flag.io.format"))
	exportCmd.Flags().String("out", "./export", i18n.T("flag.io.out"))

	importCmd.Flags().String("source", "", i18n.T("flag.io.source"))
	importCmd.Flags().String("file", "", i18n.T("flag.io.file"))
}

// --- Export ---

func exportJSON(s *store.Store, outDir string) error {
	// 联系人
	slugs, err := s.FS.ListContactFiles()
	if err != nil {
		return fmt.Errorf("list contacts: %w", err)
	}
	var contacts []*model.Contact
	for _, slug := range slugs {
		c, err := s.FS.ReadContact(slug)
		if err != nil {
			fmt.Fprintf(os.Stderr, i18n.T("output.io.skip_file")+"\n", slug, err)
			continue
		}
		c.Slug = slug
		contacts = append(contacts, c)
	}
	if err := writeJSONFile(filepath.Join(outDir, "contacts.json"), contacts); err != nil {
		return err
	}
	fmt.Printf(i18n.T("output.io.export_contacts")+"\n", len(contacts))

	// 商机
	dealFiles, err := s.FS.ListDealFiles()
	if err != nil {
		return fmt.Errorf("list deals: %w", err)
	}
	var deals []*model.Deal
	for _, f := range dealFiles {
		d, err := readDealFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, i18n.T("output.io.skip_file")+"\n", f, err)
			continue
		}
		deals = append(deals, d)
	}
	if err := writeJSONFile(filepath.Join(outDir, "deals.json"), deals); err != nil {
		return err
	}
	fmt.Printf(i18n.T("output.io.export_deals")+"\n", len(deals))

	// 活动
	activities, err := s.Activities.ListSince("", 999999)
	if err != nil {
		return fmt.Errorf("list activities: %w", err)
	}
	if err := writeJSONFile(filepath.Join(outDir, "activities.json"), activities); err != nil {
		return err
	}
	fmt.Printf(i18n.T("output.io.export_activities")+"\n", len(activities))

	// 事件
	events, err := s.Events.Poll(0, "", "", 999999)
	if err != nil {
		return fmt.Errorf("list events: %w", err)
	}
	if err := writeJSONFile(filepath.Join(outDir, "events.json"), events); err != nil {
		return err
	}
	fmt.Printf(i18n.T("output.io.export_events")+"\n", len(events))

	// 提醒
	alerts, err := s.FS.ReadPendingAlerts()
	if err != nil {
		return fmt.Errorf("read alerts: %w", err)
	}
	if err := writeJSONFile(filepath.Join(outDir, "alerts.json"), alerts); err != nil {
		return err
	}
	if len(alerts) > 0 {
		fmt.Printf(i18n.T("output.io.export_alerts")+"\n", len(alerts))
	}

	fmt.Printf(i18n.T("output.io.export.done")+"\n", outDir)
	return nil
}

func exportCSV(s *store.Store, outDir string) error {
	// 联系人 CSV
	slugs, err := s.FS.ListContactFiles()
	if err != nil {
		return fmt.Errorf("list contacts: %w", err)
	}
	csvFile, err := os.Create(filepath.Join(outDir, "contacts.csv"))
	if err != nil {
		return err
	}
	defer csvFile.Close()

	w := csv.NewWriter(csvFile)
	w.Write([]string{"id", "name", "email", "company", "title", "phone", "tags", "source", "created_at"})
	for _, slug := range slugs {
		c, err := s.FS.ReadContact(slug)
		if err != nil {
			continue
		}
		email := ""
		if len(c.Emails) > 0 {
			email = c.Emails[0]
		}
		phone := ""
		if len(c.Phones) > 0 {
			phone = c.Phones[0]
		}
		w.Write([]string{
			c.ID, c.Name, email, c.Company, c.Title,
			phone, strings.Join(c.Tags, ";"), c.Source, c.CreatedAt,
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	fmt.Printf(i18n.T("output.io.export_contacts")+"\n", len(slugs))

	// 商机 CSV
	dealFiles, err := s.FS.ListDealFiles()
	if err != nil {
		return fmt.Errorf("list deals: %w", err)
	}
	dealCSV, err := os.Create(filepath.Join(outDir, "deals.csv"))
	if err != nil {
		return err
	}
	defer dealCSV.Close()

	dw := csv.NewWriter(dealCSV)
	dw.Write([]string{"id", "title", "stage", "amount", "currency", "owner", "expected_close_at", "created_at"})
	for _, f := range dealFiles {
		d, err := readDealFile(f)
		if err != nil {
			continue
		}
		dw.Write([]string{
			d.ID, d.Title, d.Stage,
			fmt.Sprintf("%d", d.Amount), d.Currency,
			d.Owner, d.ExpectedCloseAt, d.CreatedAt,
		})
	}
	dw.Flush()
	if err := dw.Error(); err != nil {
		return err
	}
	fmt.Printf(i18n.T("output.io.export_deals")+"\n", len(dealFiles))

	// 活动 CSV
	activities, err := s.Activities.ListSince("", 999999)
	if err != nil {
		return fmt.Errorf("list activities: %w", err)
	}
	actCSV, err := os.Create(filepath.Join(outDir, "activities.csv"))
	if err != nil {
		return err
	}
	defer actCSV.Close()

	aw := csv.NewWriter(actCSV)
	aw.Write([]string{"id", "ts", "type", "direction", "channel", "summary", "contact_ids", "deal_ids"})
	for _, a := range activities {
		aw.Write([]string{
			a.ID, a.Timestamp, a.Type, a.Direction, a.Channel,
			a.Summary, strings.Join(a.ContactIDs, ";"), strings.Join(a.DealIDs, ";"),
		})
	}
	aw.Flush()
	if err := aw.Error(); err != nil {
		return err
	}
	fmt.Printf("已导出 %d 条活动\n", len(activities))

	fmt.Printf("导出完成: %s\n", outDir)
	return nil
}

// --- Import ---

func importVCard(s *store.Store, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var contacts []*model.Contact
	var current *model.Contact
	inVCard := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "BEGIN:VCARD") {
			current = model.NewContact()
			current.ID = model.ContactID()
			inVCard = true
			continue
		}
		if strings.HasPrefix(line, "END:VCARD") {
			if current != nil && current.Name != "" {
				contacts = append(contacts, current)
			}
			current = nil
			inVCard = false
			continue
		}
		if !inVCard || current == nil {
			continue
		}

		switch {
		case strings.HasPrefix(line, "FN:"):
			current.Name = strings.TrimPrefix(line, "FN:")
		case strings.HasPrefix(line, "EMAIL"):
			parts := split2(line, ":")
			if parts != nil {
				current.Emails = []string{parts[1]}
			}
		case strings.HasPrefix(line, "TEL"):
			parts := split2(line, ":")
			if parts != nil && parts[1] != "" {
				current.Phones = []string{parts[1]}
			}
		case strings.HasPrefix(line, "ORG"):
			current.Company = strings.TrimPrefix(line, "ORG:")
		case strings.HasPrefix(line, "TITLE"):
			current.Title = strings.TrimPrefix(line, "TITLE:")
		case strings.HasPrefix(line, "NOTE"):
			current.Body = strings.TrimPrefix(line, "NOTE:")
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	imported := 0
	for _, c := range contacts {
		slug := model.Slugify(c.Name)
		if slug == "" {
			slug = c.ID
		}
		c.Slug = slug
		if len(c.Slug) < 8 && c.Slug != c.ID {
			c.Slug = slug + "-" + c.ID[:8]
		}

		created, err := s.Contacts.Upsert(c)
		if err != nil {
			fmt.Fprintf(os.Stderr, i18n.T("output.io.import_fail")+"\n", c.Name, err)
			continue
		}
		if err := s.FS.WriteContact(c); err != nil {
			fmt.Fprintf(os.Stderr, i18n.T("output.io.write_fail")+"\n", c.Name, err)
			continue
		}
		eventType := model.EventContactCreated
		if !created {
			eventType = model.EventContactUpdated
		}
		s.Events.Append(getActor(), eventType, map[string]string{"id": c.ID, "name": c.Name})
		imported++
	}

	fmt.Printf(i18n.T("output.io.import_done")+"\n", imported, len(contacts))
	return nil
}

func importCSV(s *store.Store, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("read csv: %w", err)
	}
	if len(records) < 2 {
		// format string is a translated message; placeholder consistency enforced by TestPlaceholderConsistency
		return fmt.Errorf(i18n.T("error.io.csv.header_required"))
	}

	header := records[0]
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.TrimSpace(strings.ToLower(col))] = i
	}

	imported := 0
	for _, row := range records[1:] {
		c := model.NewContact()

		getCol := func(name string) string {
			if idx, ok := colMap[name]; ok && idx < len(row) {
				return strings.TrimSpace(row[idx])
			}
			return ""
		}

		name := getCol("name")
		if name == "" {
			continue
		}

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

		if email := getCol("email"); email != "" {
			c.Emails = []string{email}
		}
		c.Company = getCol("company")
		c.Title = getCol("title")
		if phone := getCol("phone"); phone != "" {
			c.Phones = []string{phone}
		}
		if tags := getCol("tags"); tags != "" {
			c.Tags = strings.Split(tags, ";")
		}
		c.Source = getCol("source")

		created, err := s.Contacts.Upsert(c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ⚠ 导入失败 %s: %v\n", c.Name, err)
			continue
		}
		if err := s.FS.WriteContact(c); err != nil {
			fmt.Fprintf(os.Stderr, "  ⚠ 写入文件失败 %s: %v\n", c.Name, err)
			continue
		}
		eventType := model.EventContactCreated
		if !created {
			eventType = model.EventContactUpdated
		}
		s.Events.Append(getActor(), eventType, map[string]string{"id": c.ID, "name": c.Name})
		imported++
	}

	fmt.Printf(i18n.T("output.io.import_done_simple")+"\n", imported)
	return nil
}

// --- 辅助 ---

func writeJSONFile(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// readDealFile 从文件系统读取商机（不经过 store）。
func readDealFile(path string) (*model.Deal, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseDealMarkdown(data), nil
}

// parseDealMarkdown 解析商机 markdown（frontmatter + body）。
func parseDealMarkdown(data []byte) *model.Deal {
	content := string(data)
	var frontmatter, body string
	if strings.HasPrefix(content, "---\n") {
		rest := content[4:]
		endIdx := strings.Index(rest, "\n---\n")
		if endIdx >= 0 {
			frontmatter = rest[:endIdx]
			body = rest[endIdx+5:]
		}
	}
	d := model.NewDeal()
	if frontmatter != "" {
		yaml.Unmarshal([]byte(frontmatter), d)
	}
	d.Body = strings.TrimSpace(body)
	return d
}
