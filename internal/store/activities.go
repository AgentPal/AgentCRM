package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AgentPal/AgentCRM/internal/model"
)

// ActivityStore 提供对活动的数据库操作。
type ActivityStore struct {
	db *DB
}

func NewActivityStore(db *DB) *ActivityStore {
	return &ActivityStore{db: db}
}

// Log 记录一条活动。如果 dedupe_key 已存在，返回现有 ID 且不创建。
func (s *ActivityStore) Log(a *model.Activity) (created bool, err error) {
	// 检查 dedupe_key 唯一性
	if a.DedupeKey != "" {
		var existingID string
		err := s.db.QueryRow(`SELECT id FROM activities WHERE dedupe_key = ?`, a.DedupeKey).Scan(&existingID)
		if err == nil {
			a.ID = existingID
			return false, nil
		}
		if err != sql.ErrNoRows {
			return false, fmt.Errorf("check dedupe: %w", err)
		}
	}

	contactIDsJSON, _ := json.Marshal(a.ContactIDs)
	dealIDsJSON, _ := json.Marshal(a.DealIDs)

	_, err = s.db.Exec(`
		INSERT INTO activities (id, ts, type, channel, direction, actor,
			contact_ids, deal_ids, summary, body, subject, dedupe_key, source_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.Timestamp, a.Type, a.Channel, a.Direction, a.Actor,
		string(contactIDsJSON), string(dealIDsJSON), a.Summary, a.Body, a.Subject,
		a.DedupeKey, a.SourceURL)
	if err != nil {
		return false, fmt.Errorf("log activity: %w", err)
	}

	// 更新 FTS 索引
	ftsBody := a.Summary
	if a.Body != "" {
		ftsBody += " " + a.Body
	}
	_, err = s.db.Exec(`INSERT INTO activities_fts (id, summary, body) VALUES (?, ?, ?)`,
		a.ID, a.Summary, ftsBody)
	if err != nil {
		fmt.Printf("FTS warning: %v\n", err)
	}

	return true, nil
}

// ListByContact 列出某联系人的活动。
func (s *ActivityStore) ListByContact(contactID string, since string, activityType string, limit int) ([]model.Activity, error) {
	if limit <= 0 {
		limit = 50
	}

	where := []string{"contact_ids LIKE ?"}
	args := []interface{}{"%" + contactID + "%"}

	if since != "" {
		where = append(where, "ts >= ?")
		args = append(args, since)
	}
	if activityType != "" {
		where = append(where, "type = ?")
		args = append(args, activityType)
	}

	query := fmt.Sprintf(`
		SELECT id, ts, type, channel, direction, actor,
			contact_ids, deal_ids, summary, body, subject, dedupe_key, source_url
		FROM activities
		WHERE %s
		ORDER BY ts DESC
		LIMIT ?`, strings.Join(where, " AND "))
	args = append(args, limit)

	return s.queryActivities(query, args...)
}

// ListSince 列出指定时间后的所有活动。
func (s *ActivityStore) ListSince(since string, limit int) ([]model.Activity, error) {
	if limit <= 0 {
		limit = 100
	}

	return s.queryActivities(`
		SELECT id, ts, type, channel, direction, actor,
			contact_ids, deal_ids, summary, body, subject, dedupe_key, source_url
		FROM activities
		WHERE ts >= ?
		ORDER BY ts DESC
		LIMIT ?`, since, limit)
}

// Search 搜索活动摘要和正文。
func (s *ActivityStore) Search(query string, limit int) ([]model.Activity, error) {
	if limit <= 0 {
		limit = 20
	}

	// FTS 搜索优先
	results, err := s.searchFTS(query, limit)
	if err == nil && len(results) > 0 {
		return results, nil
	}

	// fallback 到 LIKE（CJK 文本 FTS 可能返回空）
	return s.searchLike(query, limit)
}

func (s *ActivityStore) searchFTS(query string, limit int) ([]model.Activity, error) {
	rows, err := s.db.Query(`
		SELECT a.id, a.ts, a.type, a.channel, a.direction, a.actor,
			a.contact_ids, a.deal_ids, a.summary, a.body, a.subject, a.dedupe_key, a.source_url
		FROM activities a
		INNER JOIN activities_fts f ON a.id = f.id
		WHERE activities_fts MATCH ?
		ORDER BY rank
		LIMIT ?`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanActivities(rows)
}

func (s *ActivityStore) searchLike(query string, limit int) ([]model.Activity, error) {
	rows, err := s.db.Query(`
		SELECT id, ts, type, channel, direction, actor,
			contact_ids, deal_ids, summary, body, subject, dedupe_key, source_url
		FROM activities
		WHERE summary LIKE ? OR body LIKE ?
		ORDER BY ts DESC
		LIMIT ?`, `%`+query+`%`, `%`+query+`%`, limit)
	if err != nil {
		return nil, fmt.Errorf("search activities: %w", err)
	}
	defer rows.Close()
	return scanActivities(rows)
}

func (s *ActivityStore) queryActivities(query string, args ...interface{}) ([]model.Activity, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query activities: %w", err)
	}
	defer rows.Close()
	return scanActivities(rows)
}

func scanActivities(rows *sql.Rows) ([]model.Activity, error) {
	var activities []model.Activity
	for rows.Next() {
		var a model.Activity
		var contactIDsJSON, dealIDsJSON string
		if err := rows.Scan(&a.ID, &a.Timestamp, &a.Type, &a.Channel, &a.Direction, &a.Actor,
			&contactIDsJSON, &dealIDsJSON, &a.Summary, &a.Body, &a.Subject,
			&a.DedupeKey, &a.SourceURL); err != nil {
			return nil, fmt.Errorf("scan activity: %w", err)
		}
		json.Unmarshal([]byte(contactIDsJSON), &a.ContactIDs)
		json.Unmarshal([]byte(dealIDsJSON), &a.DealIDs)
		activities = append(activities, a)
	}
	return activities, nil
}
