package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/agentcrm/agentcrm/internal/model"
)

// ContactStore 提供对联系人索引的数据库操作。
type ContactStore struct {
	db *DB
}

func NewContactStore(db *DB) *ContactStore {
	return &ContactStore{db: db}
}

// Upsert 按 email 去重插入或更新联系人。
func (s *ContactStore) Upsert(c *model.Contact) (created bool, err error) {
	emailsJSON, _ := json.Marshal(c.Emails)
	tagsJSON, _ := json.Marshal(c.Tags)

	// 按第一个 email 查找
	var existingID string
	if len(c.Emails) > 0 && c.Emails[0] != "" {
		err = s.db.QueryRow(
			`SELECT id FROM contacts WHERE emails LIKE ?`,
			`%`+c.Emails[0]+`%`,
		).Scan(&existingID)
		if err != nil && err != sql.ErrNoRows {
			return false, fmt.Errorf("check existing: %w", err)
		}
	}

	if existingID != "" {
		_, err = s.db.Exec(`
			UPDATE contacts SET
				name = COALESCE(NULLIF(?, ''), name),
				company = COALESCE(NULLIF(?, ''), company),
				title = COALESCE(NULLIF(?, ''), title),
				emails = ?,
				tags = ?,
				updated_at = ?,
				source = COALESCE(NULLIF(?, ''), source)
			WHERE id = ?`,
			c.Name, c.Company, c.Title, string(emailsJSON),
			string(tagsJSON), c.UpdatedAt, c.Source, existingID,
		)
		if err != nil {
			return false, fmt.Errorf("update contact: %w", err)
		}
		c.ID = existingID
		return false, nil
	}

	// 插入
	if c.Slug == "" {
		c.Slug = model.Slugify(c.Name)
	}
	if c.Slug == "" {
		c.Slug = c.ID
	}

	hasBirthday := 0
	if c.Birthday != "" {
		hasBirthday = 1
	}

	_, err = s.db.Exec(`
		INSERT INTO contacts (id, slug, name, emails, company, title, tags,
			last_activity_at, updated_at, has_birthday,
			social_twitter, social_linkedin, social_wechat, social_github,
			source, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Slug, c.Name, string(emailsJSON),
		c.Company, c.Title, string(tagsJSON),
		c.LastActivityAt, c.UpdatedAt, hasBirthday,
		c.Social.Twitter, c.Social.LinkedIn, c.Social.WeChat, c.Social.GitHub,
		c.Source, c.CreatedAt,
	)
	if err != nil {
		return false, fmt.Errorf("insert contact: %w", err)
	}

	// 更新 FTS 索引
	body := c.Body
	if body == "" {
		body = c.Name + " " + c.Company
	}
	_, err = s.db.Exec(`INSERT INTO contacts_fts (id, name, company, body) VALUES (?, ?, ?, ?)`,
		c.ID, c.Name, c.Company, body)
	if err != nil {
		fmt.Printf("FTS warning: %v\n", err)
	}

	return true, nil
}

// GetByID 通过 ID 获取联系人。
func (s *ContactStore) GetByID(id string) (*model.Contact, error) {
	return s.scanContact(s.db.QueryRow(`
		SELECT id, slug, name, emails, company, title, tags,
			last_activity_at, updated_at, has_birthday,
			social_twitter, social_linkedin, social_wechat, social_github,
			source, created_at
		FROM contacts WHERE id = ?`, id))
}

// GetByEmail 通过 email 查询联系人。
func (s *ContactStore) GetByEmail(email string) (*model.Contact, error) {
	return s.scanContact(s.db.QueryRow(`
		SELECT id, slug, name, emails, company, title, tags,
			last_activity_at, updated_at, has_birthday,
			social_twitter, social_linkedin, social_wechat, social_github,
			source, created_at
		FROM contacts WHERE emails LIKE ?`, `%`+email+`%`))
}

// Search 通过 FTS 或 LIKE 搜索联系人。
func (s *ContactStore) Search(query string, limit int) ([]model.ContactSearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	// FTS 搜索
	results, err := s.searchFTS(query, limit)
	if err == nil && len(results) > 0 {
		return results, nil
	}

	// fallback 到 LIKE（CJK 文本 FTS 可能返回空）
	return s.searchLike(query, limit)
}

func (s *ContactStore) searchFTS(query string, limit int) ([]model.ContactSearchResult, error) {
	rows, err := s.db.Query(`
		SELECT c.id, c.name, c.company, c.title, c.emails, c.last_activity_at, c.tags
		FROM contacts c
		INNER JOIN contacts_fts f ON c.id = f.id
		WHERE contacts_fts MATCH ?
		ORDER BY rank
		LIMIT ?`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSearchResults(rows)
}

func (s *ContactStore) searchLike(query string, limit int) ([]model.ContactSearchResult, error) {
	rows, err := s.db.Query(`
		SELECT id, name, company, title, emails, last_activity_at, tags
		FROM contacts
		WHERE name LIKE ? OR company LIKE ? OR emails LIKE ?
		LIMIT ?`,
		`%`+query+`%`, `%`+query+`%`, `%`+query+`%`, limit)
	if err != nil {
		return nil, fmt.Errorf("search contacts: %w", err)
	}
	defer rows.Close()
	return scanSearchResults(rows)
}

// List 列出符合条件的联系人。
func (s *ContactStore) List(tag, updatedSince, hasField string, limit int) ([]model.ContactSearchResult, error) {
	if limit <= 0 {
		limit = 50
	}

	where := []string{"1=1"}
	args := []interface{}{}

	if tag != "" {
		where = append(where, "tags LIKE ?")
		args = append(args, `%`+tag+`%`)
	}
	if updatedSince != "" {
		where = append(where, "updated_at >= ?")
		args = append(args, updatedSince)
	}
	if hasField != "" {
		switch hasField {
		case "birthday":
			where = append(where, "has_birthday = 1")
		}
	}

	query := fmt.Sprintf(`
		SELECT id, name, company, title, emails, last_activity_at, tags
		FROM contacts WHERE %s
		ORDER BY updated_at DESC
		LIMIT ?`, strings.Join(where, " AND "))
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list contacts: %w", err)
	}
	defer rows.Close()

	return scanSearchResults(rows)
}

// Update 更新联系人字段并返回旧值。
func (s *ContactStore) Update(id, field, value, reason string) (oldValue string, err error) {
	row := s.db.QueryRow(fmt.Sprintf(`SELECT %s FROM contacts WHERE id = ?`, field), id)
	if err := row.Scan(&oldValue); err != nil {
		return "", fmt.Errorf("get current value: %w", err)
	}

	if oldValue == value {
		return oldValue, nil
	}

	_, err = s.db.Exec(fmt.Sprintf(`UPDATE contacts SET %s = ?, updated_at = ? WHERE id = ?`,
		field), value, Now(), id)
	if err != nil {
		return "", fmt.Errorf("update contact: %w", err)
	}

	return oldValue, nil
}

// Delete 删除联系人索引。
func (s *ContactStore) Delete(id string) error {
	_, err := s.db.Exec(`DELETE FROM contacts WHERE id = ?`, id)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM contacts_fts WHERE id = ?`, id)
	return err
}

func (s *ContactStore) scanContact(row *sql.Row) (*model.Contact, error) {
	var c model.Contact
	var emailsJSON, tagsJSON string
	err := row.Scan(&c.ID, &c.Slug, &c.Name, &emailsJSON, &c.Company, &c.Title, &tagsJSON,
		&c.LastActivityAt, &c.UpdatedAt, &c.HasBirthday,
		&c.Social.Twitter, &c.Social.LinkedIn, &c.Social.WeChat, &c.Social.GitHub,
		&c.Source, &c.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("contact not found")
		}
		return nil, fmt.Errorf("scan contact: %w", err)
	}
	json.Unmarshal([]byte(emailsJSON), &c.Emails)
	json.Unmarshal([]byte(tagsJSON), &c.Tags)
	return &c, nil
}

func scanSearchResults(rows *sql.Rows) ([]model.ContactSearchResult, error) {
	var results []model.ContactSearchResult
	for rows.Next() {
		var r model.ContactSearchResult
		var emailsJSON, tagsJSON string
		if err := rows.Scan(&r.ID, &r.Name, &r.Company, &r.Title,
			&emailsJSON, &r.LastActivityAt, &tagsJSON); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}
		var emails []string
		json.Unmarshal([]byte(emailsJSON), &emails)
		if len(emails) > 0 {
			r.Email = emails[0]
		}
		var tags []string
		json.Unmarshal([]byte(tagsJSON), &tags)
		r.Tags = strings.Join(tags, ", ")
		results = append(results, r)
	}
	return results, nil
}
