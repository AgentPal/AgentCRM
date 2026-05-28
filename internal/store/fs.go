package store

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/agentcrm/agentcrm/internal/model"
	"gopkg.in/yaml.v3"
)

// FileStore 管理所有文件系统的读写操作。
// 所有文件操作通过此层，确保原子性和一致性。
type FileStore struct {
	rootDir string
	mu      sync.Mutex // 文件锁，串行化同一目录的写操作
}

// NewFileStore 创建文件存储层。
// rootDir 是 ~/.agentcrm/ 目录。
func NewFileStore(rootDir string) *FileStore {
	return &FileStore{rootDir: rootDir}
}

// RootDir 返回数据根目录。
func (fs *FileStore) RootDir() string {
	return fs.rootDir
}

// Init 创建完整的目录结构。
func (fs *FileStore) Init() error {
	dirs := []string{
		filepath.Join(fs.rootDir, "contacts"),
		filepath.Join(fs.rootDir, "deals"),
		filepath.Join(fs.rootDir, "activities"),
		filepath.Join(fs.rootDir, "memory", "contacts"),
		filepath.Join(fs.rootDir, "memory", "deals"),
		filepath.Join(fs.rootDir, "events"),
		filepath.Join(fs.rootDir, "subscribers"),
		filepath.Join(fs.rootDir, "proposals"),
		filepath.Join(fs.rootDir, "alerts"),
		filepath.Join(fs.rootDir, "rules"),
		filepath.Join(fs.rootDir, ".audit"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	return nil
}

// --- Contact 文件操作 ---

// ContactPath 返回联系人文件的路径。
func (fs *FileStore) ContactPath(slug string) string {
	return filepath.Join(fs.rootDir, "contacts", slug+".md")
}

// WriteContact 将联系人写入 markdown 文件（原子写入）。
func (fs *FileStore) WriteContact(c *model.Contact) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	var buf bytes.Buffer
	buf.WriteString("---\n")

	// 序列化 frontmatter
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("yaml encode contact: %w", err)
	}
	encoder.Close()

	buf.WriteString("---\n")
	if c.Body != "" {
		buf.WriteString(c.Body)
	}

	return fs.atomicWrite(fs.ContactPath(c.Slug), buf.Bytes())
}

// ReadContact 从 markdown 文件读取联系人。
func (fs *FileStore) ReadContact(slug string) (*model.Contact, error) {
	path := fs.ContactPath(slug)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read contact %s: %w", slug, err)
	}

	return parseContactMarkdown(data)
}

// DeleteContact 删除联系人文件（改名而非真删）。
func (fs *FileStore) DeleteContact(slug string, mergedInto string) error {
	oldPath := fs.ContactPath(slug)
	newPath := filepath.Join(fs.rootDir, "contacts", slug+".merged-into-"+mergedInto+".md")
	return os.Rename(oldPath, newPath)
}

// UpdateContactField 更新联系人文件的单个字段，自动归档旧值到 _history。
// 支持的字段：company, title, phone, location 等。
func (fs *FileStore) UpdateContactField(slug string, field, newValue, reason, actor string) (oldValue string, err error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	c, err := fs.ReadContact(slug)
	if err != nil {
		return "", fmt.Errorf("read contact: %w", err)
	}
	c.Slug = slug // Slug is yaml:"-", restore it for write path

	now := time.Now().UTC().Format(time.RFC3339)
	dateNow := time.Now().UTC().Format("2006-01-02")

	// 获取旧值并更新
	switch field {
	case "company":
		oldValue = c.Company
		if oldValue == newValue {
			return oldValue, nil
		}
		if oldValue != "" {
			c.CompanyHistory = append(c.CompanyHistory, model.NewFieldHistory(oldValue, dateNow, actor, reason))
			// 将上一条当前值的 to 设为现在
			for i := len(c.CompanyHistory) - 1; i >= 0; i-- {
				if c.CompanyHistory[i].IsCurrent() {
					c.CompanyHistory[i].To = dateNow
					break
				}
			}
		}
		c.Company = newValue
	case "title":
		oldValue = c.Title
		if oldValue == newValue {
			return oldValue, nil
		}
		if oldValue != "" {
			c.TitleHistory = append(c.TitleHistory, model.NewFieldHistory(oldValue, dateNow, actor, reason))
			for i := len(c.TitleHistory) - 1; i >= 0; i-- {
				if c.TitleHistory[i].IsCurrent() {
					c.TitleHistory[i].To = dateNow
					break
				}
			}
		}
		c.Title = newValue
	default:
		// 其他字段直接覆盖
		return "", fmt.Errorf("unsupported field for history archival: %s", field)
	}

	c.UpdatedAt = now

	if err := fs.writeContactFile(c); err != nil {
		return "", err
	}
	return oldValue, nil
}

// MergeContactFiles 合并两个联系人文件。keeper 优先保留，mergee 补充空缺字段。
func (fs *FileStore) MergeContactFiles(keeperID, mergeeID, keeperSlug, mergeeSlug, reason, actor string) error {
	keeper, err := fs.ReadContact(keeperSlug)
	if err != nil {
		return fmt.Errorf("read keeper: %w", err)
	}
	keeper.Slug = keeperSlug // Slug is yaml:"-", restore for write path

	mergee, err := fs.ReadContact(mergeeSlug)
	if err != nil {
		return fmt.Errorf("read mergee: %w", err)
	}
	mergee.Slug = mergeeSlug

	now := time.Now().UTC().Format(time.RFC3339)

	// 合并字段：keeper 优先，mergee 补充
	if mergee.Company != "" && keeper.Company == "" {
		keeper.Company = mergee.Company
	}
	if mergee.Title != "" && keeper.Title == "" {
		keeper.Title = mergee.Title
	}
	if mergee.Birthday != "" && keeper.Birthday == "" {
		keeper.Birthday = mergee.Birthday
	}
	if len(mergee.Emails) > 0 && len(keeper.Emails) == 0 {
		keeper.Emails = mergee.Emails
	}
	if len(mergee.Phones) > 0 && len(keeper.Phones) == 0 {
		keeper.Phones = mergee.Phones
	}
	if mergee.Body != "" && keeper.Body == "" {
		keeper.Body = mergee.Body
	}
	// 合并 tags
	existing := make(map[string]bool)
	for _, t := range keeper.Tags {
		existing[t] = true
	}
	for _, t := range mergee.Tags {
		if !existing[t] {
			keeper.Tags = append(keeper.Tags, t)
		}
	}

	keeper.UpdatedAt = now

	if err := fs.writeContactFile(keeper); err != nil {
		return fmt.Errorf("write keeper: %w", err)
	}

	// 标记 mergee 为已合并
	return fs.DeleteContact(mergeeSlug, keeperID)
}

// writeContactFile 写入联系人文件（调用方需持有锁）。
func (fs *FileStore) writeContactFile(c *model.Contact) error {
	var buf bytes.Buffer
	buf.WriteString("---\n")

	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("yaml encode: %w", err)
	}
	encoder.Close()

	buf.WriteString("---\n")
	if c.Body != "" {
		buf.WriteString(c.Body)
	}

	return fs.atomicWrite(fs.ContactPath(c.Slug), buf.Bytes())
}

// --- Deal 文件操作 ---

// DealPath 返回商机文件的路径。
func (fs *FileStore) DealPath(slug string) string {
	return filepath.Join(fs.rootDir, "deals", slug+".md")
}

// WriteDeal 将商机写入 markdown 文件。
func (fs *FileStore) WriteDeal(d *model.Deal) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.writeDealFile(d)
}

// ReadDeal 从 markdown 文件读取商机。
func (fs *FileStore) ReadDeal(slug string) (*model.Deal, error) {
	data, err := os.ReadFile(fs.DealPath(slug))
	if err != nil {
		return nil, fmt.Errorf("read deal %s: %w", slug, err)
	}
	return parseDealMarkdown(data)
}

// UpdateDealField 更新商机文件的单个字段，自动归档旧值到 _history。
func (fs *FileStore) UpdateDealField(slug string, field string, newValue string, oldValue string, reason, actor string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	d, err := fs.ReadDeal(slug)
	if err != nil {
		return fmt.Errorf("read deal: %w", err)
	}
	d.Slug = slug

	now := time.Now().UTC().Format(time.RFC3339)
	dateNow := time.Now().UTC().Format("2006-01-02")

	switch field {
	case "stage":
		if oldValue != "" && oldValue != newValue {
			d.StageHistory = append(d.StageHistory, model.StageHistory{
				Stage:     newValue,
				EnteredAt: dateNow,
				By:        actor,
				Reason:    reason,
			})
		}
		d.Stage = newValue
	case "amount":
		var newAmt, oldAmt int
		fmt.Sscanf(newValue, "%d", &newAmt)
		fmt.Sscanf(oldValue, "%d", &oldAmt)
		if oldAmt != 0 && oldAmt != newAmt {
			d.AmountHistory = append(d.AmountHistory, model.AmountHistory{
				Value:  oldAmt,
				From:   dateNow,
				To:     "~",
				Reason: reason,
			})
			for i := len(d.AmountHistory) - 1; i >= 0; i-- {
				if d.AmountHistory[i].IsCurrent() && d.AmountHistory[i].Value != newAmt {
					d.AmountHistory[i].To = dateNow
					break
				}
			}
		}
		d.Amount = newAmt
	default:
		return fmt.Errorf("unsupported deal field: %s", field)
	}

	d.UpdatedAt = now

	return fs.writeDealFile(d)
}

// DeleteDeal 删除商机文件。
func (fs *FileStore) DeleteDeal(slug string) error {
	path := fs.DealPath(slug)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// writeDealFile 写入商机文件（调用方需持有锁）。
func (fs *FileStore) writeDealFile(d *model.Deal) error {
	var buf bytes.Buffer
	buf.WriteString("---\n")

	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(d); err != nil {
		return fmt.Errorf("yaml encode deal: %w", err)
	}
	encoder.Close()

	buf.WriteString("---\n")
	if d.Body != "" {
		buf.WriteString(d.Body)
	}

	return fs.atomicWrite(fs.DealPath(d.Slug), buf.Bytes())
}

// parseDealMarkdown 解析商机 markdown（frontmatter + body）。
func parseDealMarkdown(data []byte) (*model.Deal, error) {
	content := string(data)

	var frontmatter string
	var body string

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

// --- Activity JSONL ---

// ActivityPath 返回当前月份的活动文件路径。
func (fs *FileStore) ActivityPath(timestamp string) string {
	month := timestamp[:7] // "2026-05"
	return filepath.Join(fs.rootDir, "activities", month+".jsonl")
}

// AppendActivity 追加一条活动到 JSONL 文件。
func (fs *FileStore) AppendActivity(a *model.Activity) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	line, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("json marshal activity: %w", err)
	}

	path := fs.ActivityPath(a.Timestamp)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open activity file: %w", err)
	}
	defer f.Close()

	_, err = f.Write(append(line, '\n'))
	return err
}

// ReadActivities 读取一个月内的所有活动。
func (fs *FileStore) ReadActivities(timestamp string) ([]model.Activity, error) {
	path := fs.ActivityPath(timestamp)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open activities: %w", err)
	}
	defer f.Close()

	var activities []model.Activity
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var a model.Activity
		if err := json.Unmarshal(line, &a); err != nil {
			continue // 跳过损坏行
		}
		activities = append(activities, a)
	}
	return activities, scanner.Err()
}

// --- Event JSONL ---

// EventPath 返回事件文件路径。
func (fs *FileStore) EventPath(timestamp string) string {
	month := timestamp[:7]
	return filepath.Join(fs.rootDir, "events", month+".jsonl")
}

// AppendEvent 追加事件到 JSONL。
func (fs *FileStore) AppendEvent(e *model.Event) error {
	line, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("json marshal event: %w", err)
	}

	path := fs.EventPath(e.TS)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open event file: %w", err)
	}
	defer f.Close()

	_, err = f.Write(append(line, '\n'))
	return err
}

// --- Memory 文件 ---

// MemoryContactPath 返回联系人记忆文件路径。
func (fs *FileStore) MemoryContactPath(contactID string) string {
	return filepath.Join(fs.rootDir, "memory", "contacts", contactID+".md")
}

// WriteContactMemory 写入记忆文件。
func (fs *FileStore) WriteContactMemory(contactID string, content string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.atomicWrite(fs.MemoryContactPath(contactID), []byte(content))
}

// ReadContactMemory 读取记忆文件。
func (fs *FileStore) ReadContactMemory(contactID string) (string, error) {
	data, err := os.ReadFile(fs.MemoryContactPath(contactID))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// --- Config ---

// ConfigPath 返回配置文件路径。
func (fs *FileStore) ConfigPath() string {
	return filepath.Join(fs.rootDir, "config.json")
}

// WriteConfig 写入配置。
func (fs *FileStore) WriteConfig(cfg *model.Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return fs.atomicWrite(fs.ConfigPath(), data)
}

// ReadConfig 读取配置。
func (fs *FileStore) ReadConfig() (*model.Config, error) {
	data, err := os.ReadFile(fs.ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return model.DefaultConfig(fs.rootDir), nil
		}
		return nil, err
	}
	var cfg model.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.DataDir = fs.rootDir
	return &cfg, nil
}

// --- Subscriber Cursor ---

// SubscriberPath 返回订阅者 cursor 文件路径。
func (fs *FileStore) SubscriberPath(actor string) string {
	return filepath.Join(fs.rootDir, "subscribers", actor+".json")
}

// WriteSubscriberCursor 写入 cursor。
func (fs *FileStore) WriteSubscriberCursor(sc *model.SubscriberCursor) error {
	data, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		return err
	}
	return fs.atomicWrite(fs.SubscriberPath(sc.Actor), data)
}

// ReadSubscriberCursor 读取 cursor。
func (fs *FileStore) ReadSubscriberCursor(actor string) (*model.SubscriberCursor, error) {
	data, err := os.ReadFile(fs.SubscriberPath(actor))
	if err != nil {
		if os.IsNotExist(err) {
			return &model.SubscriberCursor{Actor: actor, LastSeq: 0}, nil
		}
		return nil, err
	}
	var sc model.SubscriberCursor
	if err := json.Unmarshal(data, &sc); err != nil {
		return nil, fmt.Errorf("parse subscriber cursor: %w", err)
	}
	return &sc, nil
}

// --- Proposals ---

// ProposalsPath 返回 proposals 文件路径。
func (fs *FileStore) ProposalsPath() string {
	return filepath.Join(fs.rootDir, "proposals", "pending.jsonl")
}

// AppendProposal 追加 proposal。
func (fs *FileStore) AppendProposal(p *model.Proposal) error {
	line, err := json.Marshal(p)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(fs.ProposalsPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}

// --- Alerts ---

// PendingAlertsPath 返回 pending alerts 路径。
func (fs *FileStore) PendingAlertsPath() string {
	return filepath.Join(fs.rootDir, "alerts", "pending.json")
}

// WritePendingAlerts 写入 pending alerts。
func (fs *FileStore) WritePendingAlerts(alerts []model.Alert) error {
	data, err := json.MarshalIndent(alerts, "", "  ")
	if err != nil {
		return err
	}
	return fs.atomicWrite(fs.PendingAlertsPath(), data)
}

// ReadPendingAlerts 读取 pending alerts。
func (fs *FileStore) ReadPendingAlerts() ([]model.Alert, error) {
	data, err := os.ReadFile(fs.PendingAlertsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var alerts []model.Alert
	if err := json.Unmarshal(data, &alerts); err != nil {
		return nil, err
	}
	return alerts, nil
}

// --- 内部辅助 ---

// atomicWrite 原子写文件：先写临时文件，再 rename。
func (fs *FileStore) atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)

	tmpFile, err := os.CreateTemp(dir, "tmp-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	tmpFile.Close()

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

// parseContactMarkdown 解析联系人 markdown（frontmatter + body）。
func parseContactMarkdown(data []byte) (*model.Contact, error) {
	content := string(data)

	// 提取 frontmatter (---\n...\n---\n)
	var frontmatter string
	var body string

	if strings.HasPrefix(content, "---\n") {
		rest := content[4:]
		endIdx := strings.Index(rest, "\n---\n")
		if endIdx >= 0 {
			frontmatter = rest[:endIdx]
			body = rest[endIdx+5:]
		}
	}

	c := model.NewContact()
	if frontmatter != "" {
		if err := yaml.Unmarshal([]byte(frontmatter), c); err != nil {
			return nil, fmt.Errorf("parse frontmatter: %w", err)
		}
	}
	c.Body = strings.TrimSpace(body)
	return c, nil
}

// ListDealFiles 列出所有商机文件。
func (fs *FileStore) ListDealFiles() ([]string, error) {
	dir := filepath.Join(fs.rootDir, "deals")
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

// ListActivityFiles 列出所有活动文件。
func (fs *FileStore) ListActivityFiles() ([]string, error) {
	dir := filepath.Join(fs.rootDir, "activities")
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

// ListContactFiles 列出所有联系人文件。
func (fs *FileStore) ListContactFiles() ([]string, error) {
	dir := filepath.Join(fs.rootDir, "contacts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var slugs []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			slug := strings.TrimSuffix(e.Name(), ".md")
			if !strings.Contains(slug, ".merged-into-") {
				slugs = append(slugs, slug)
			}
		}
	}
	return slugs, nil
}
