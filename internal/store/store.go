package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agentcrm/agentcrm/internal/model"
	"gopkg.in/yaml.v3"
)

// Store 聚合 DB 和 FileStore，是 CLI 命令的主要数据访问入口。
type Store struct {
	DB        *DB
	FS        *FileStore
	Contacts  *ContactStore
	Deals     *DealStore
	Activities *ActivityStore
	Events    *EventStore
	Memos     *MemoStore

	configDir string // ~/.agentcrm
}

// Open 打开或创建数据目录和数据库。
func Open(configDir string) (*Store, error) {
	// 展开 ~
	if len(configDir) > 0 && configDir[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("home dir: %w", err)
		}
		configDir = filepath.Join(home, configDir[1:])
	}

	absDir, err := filepath.Abs(configDir)
	if err != nil {
		return nil, fmt.Errorf("abs path: %w", err)
	}

	// 确保目录存在
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	// 打开 SQLite
	dbPath := filepath.Join(absDir, "index.db")
	db, err := OpenDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	fs := NewFileStore(absDir)

	s := &Store{
		DB:         db,
		FS:         fs,
		Contacts:   NewContactStore(db),
		Deals:      NewDealStore(db),
		Activities: NewActivityStore(db),
		Events:     NewEventStore(db),
		Memos:      NewMemoStore(db, absDir),
		configDir:  absDir,
	}

	return s, nil
}

// Close 关闭数据库连接。
func (s *Store) Close() error {
	return s.DB.Close()
}

// ConfigDir 返回配置目录。
func (s *Store) ConfigDir() string {
	return s.configDir
}

// Reindex 从所有文件重建 SQLite 索引。
func (s *Store) Reindex() error {
	// 1. 清空所有表
	s.DB.Exec("DELETE FROM contacts")
	s.DB.Exec("DELETE FROM contacts_fts")
	s.DB.Exec("DELETE FROM deals")
	s.DB.Exec("DELETE FROM activities")
	s.DB.Exec("DELETE FROM activities_fts")
	s.DB.Exec("DELETE FROM events")

	// 2. 重建联系人索引
	contactSlugs, err := s.FS.ListContactFiles()
	if err != nil {
		return fmt.Errorf("list contact files: %w", err)
	}
	for _, slug := range contactSlugs {
		c, err := s.FS.ReadContact(slug)
		if err != nil {
			fmt.Printf("  ⚠ 跳过 %s: %v\n", slug, err)
			continue
		}
		c.Slug = slug

		if c.ID == "" {
			c.ID = model.ContactID()
		}
		if c.CreatedAt == "" {
			c.CreatedAt = Now()
		}
		if c.UpdatedAt == "" {
			c.UpdatedAt = Now()
		}
		if len(c.Emails) == 0 {
			c.Emails = []string{""}
		}

		emailsJSON, _ := json.Marshal(c.Emails)
		tagsJSON, _ := json.Marshal(c.Tags)

		hasBirthday := 0
		if c.Birthday != "" {
			hasBirthday = 1
		}

		_, err = s.DB.Exec(`
			INSERT OR IGNORE INTO contacts (id, slug, name, emails, company, title, tags,
				last_activity_at, updated_at, has_birthday,
				social_twitter, social_linkedin, social_wechat, social_github,
				source, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			c.ID, slug, c.Name, string(emailsJSON),
			c.Company, c.Title, string(tagsJSON),
			c.LastActivityAt, c.UpdatedAt, hasBirthday,
			c.Social.Twitter, c.Social.LinkedIn, c.Social.WeChat, c.Social.GitHub,
			c.Source, c.CreatedAt)

		if err != nil {
			fmt.Printf("  ⚠ 写入联系人 %s 失败: %v\n", slug, err)
			continue
		}

		// FTS
		ftsBody := c.Name + " " + c.Company
		s.DB.Exec(`INSERT OR IGNORE INTO contacts_fts (id, name, company, body) VALUES (?, ?, ?, ?)`,
			c.ID, c.Name, c.Company, ftsBody)
	}

	// 3. 重建活动索引
	activityFiles, err := s.listActivityFiles()
	if err != nil {
		return fmt.Errorf("list activity files: %w", err)
	}
	for _, file := range activityFiles {
		activities, err := readActivityFile(file)
		if err != nil {
			fmt.Printf("  ⚠ 跳过活动文件 %s: %v\n", file, err)
			continue
		}
		for _, a := range activities {
			contactIDsJSON, _ := json.Marshal(a.ContactIDs)
			dealIDsJSON, _ := json.Marshal(a.DealIDs)

			_, err := s.DB.Exec(`
				INSERT OR IGNORE INTO activities (id, ts, type, channel, direction, actor,
					contact_ids, deal_ids, summary, body, subject, dedupe_key, source_url)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				a.ID, a.Timestamp, a.Type, a.Channel, a.Direction, a.Actor,
				string(contactIDsJSON), string(dealIDsJSON), a.Summary, a.Body, a.Subject,
				a.DedupeKey, a.SourceURL)
			if err != nil {
				fmt.Printf("  ⚠ 写入活动 %s 失败: %v\n", a.ID, err)
				continue
			}

			// FTS
			ftsBody := a.Summary
			if a.Body != "" {
				ftsBody += " " + a.Body
			}
			s.DB.Exec(`INSERT OR IGNORE INTO activities_fts (id, summary, body) VALUES (?, ?, ?)`,
				a.ID, a.Summary, ftsBody)
		}
	}

	// 4. 重建商机索引
	dealFiles, err := s.listDealFiles()
	if err != nil {
		return fmt.Errorf("list deal files: %w", err)
	}
	for _, file := range dealFiles {
		slug := strings.TrimSuffix(filepath.Base(file), ".md")
		d, err := parseDealMarkdownFile(file)
		if err != nil {
			fmt.Printf("  ⚠ 跳过商机 %s: %v\n", slug, err)
			continue
		}
		d.Slug = slug

		if d.ID == "" {
			d.ID = model.DealID()
		}
		if d.CreatedAt == "" {
			d.CreatedAt = Now()
		}
		if d.UpdatedAt == "" {
			d.UpdatedAt = Now()
		}

		contactIDsJSON, _ := json.Marshal(d.ContactIDs)

		_, err = s.DB.Exec(`
			INSERT OR IGNORE INTO deals (id, slug, title, stage, amount, currency,
				contact_ids, expected_close_at, owner, updated_at, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			d.ID, slug, d.Title, d.Stage, d.Amount, d.Currency,
			string(contactIDsJSON), d.ExpectedCloseAt, d.Owner, d.UpdatedAt, d.CreatedAt)
		if err != nil {
			fmt.Printf("  ⚠ 写入商机 %s 失败: %v\n", slug, err)
			continue
		}
	}

	fmt.Printf("重建完成: %d 联系人, %d 商机\n", len(contactSlugs), len(dealFiles))
	return nil
}

// readActivityFile 读取一个 JSONL 活动文件中的所有活动。
func readActivityFile(path string) ([]model.Activity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var activities []model.Activity
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var a model.Activity
		if err := json.Unmarshal([]byte(line), &a); err != nil {
			continue // 跳过损坏行
		}
		activities = append(activities, a)
	}
	return activities, nil
}

// parseDealMarkdownFile 解析商机 markdown 文件。
func parseDealMarkdownFile(path string) (*model.Deal, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
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
		if err := yaml.Unmarshal([]byte(frontmatter), d); err != nil {
			return nil, fmt.Errorf("parse deal frontmatter: %w", err)
		}
	}
	d.Body = strings.TrimSpace(body)
	return d, nil
}

func (s *Store) listActivityFiles() ([]string, error) {
	dir := filepath.Join(s.configDir, "activities")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files, nil
}

func (s *Store) listDealFiles() ([]string, error) {
	dir := filepath.Join(s.configDir, "deals")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files, nil
}
