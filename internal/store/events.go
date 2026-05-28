package store

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AgentPal/AgentCRM/internal/model"
)

// EventStore 提供对事件日志的数据库操作。
type EventStore struct {
	db *DB
}

func NewEventStore(db *DB) *EventStore {
	return &EventStore{db: db}
}

// Append 追加一条事件并返回 seq。
func (s *EventStore) Append(actor, eventType string, payload interface{}) (int64, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal payload: %w", err)
	}

	result, err := s.db.Exec(`
		INSERT INTO events (ts, actor, type, payload)
		VALUES (?, ?, ?, ?)`, Now(), actor, eventType, string(payloadJSON))
	if err != nil {
		return 0, fmt.Errorf("append event: %w", err)
	}

	seq, _ := result.LastInsertId()
	return seq, nil
}

// Poll 拉取自某个 seq 之后的新事件。
func (s *EventStore) Poll(afterSeq int64, filter string, excludeActor string, limit int) ([]model.Event, error) {
	if limit <= 0 {
		limit = 100
	}

	where := []string{"seq > ?"}
	args := []interface{}{afterSeq}

	if filter != "" {
		// filter 格式: "type=contact.created" 或 "type=activity.logged AND channel=twitter"
		conditions := strings.Split(filter, " AND ")
		for _, cond := range conditions {
			cond = strings.TrimSpace(cond)
			parts := strings.SplitN(cond, "=", 2)
			if len(parts) == 2 {
				where = append(where, fmt.Sprintf("%s LIKE ?", parts[0]))
				args = append(args, parts[1])
			}
		}
	}

	if excludeActor != "" {
		where = append(where, "actor != ?")
		args = append(args, excludeActor)
	}

	query := fmt.Sprintf(`
		SELECT seq, ts, actor, type, payload
		FROM events
		WHERE %s
		ORDER BY seq ASC
		LIMIT ?`, strings.Join(where, " AND "))
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("poll events: %w", err)
	}
	defer rows.Close()

	var events []model.Event
	for rows.Next() {
		var e model.Event
		if err := rows.Scan(&e.Seq, &e.TS, &e.Actor, &e.Type, &e.Payload); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, e)
	}
	return events, nil
}

// GetLastSeq 返回当前最大 seq。
func (s *EventStore) GetLastSeq() (int64, error) {
	var seq sql.NullInt64
	err := s.db.QueryRow(`SELECT MAX(seq) FROM events`).Scan(&seq)
	if err != nil {
		return 0, err
	}
	return seq.Int64, nil
}

// MemoStore 提供对记忆的数据库操作。
type MemoStore struct {
	db       *DB
	configDir string
}

func NewMemoStore(db *DB, configDir string) *MemoStore {
	return &MemoStore{db: db, configDir: configDir}
}

// Insert 插入一条 memo。
func (s *MemoStore) Insert(m *model.Memo) error {
	now := Now()
	_, err := s.db.Exec(`
		INSERT INTO memos (id, scope_type, scope_id, text, valid_from, valid_to,
			decay_policy, decayed_at, source, actor, created_at, expired)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.ScopeType, m.ScopeID, m.Text, m.ValidFrom, m.ValidTo,
		m.Decay, "", m.Source, m.Actor, now, false)
	return err
}

// ListByScope 列出某 scope 的所有 memo。
// scopeType="" 时列出所有 memo。
func (s *MemoStore) ListByScope(scopeType, scopeID string) ([]model.Memo, error) {
	where := []string{"1=1"}
	args := []interface{}{}

	if scopeType != "" {
		where = append(where, "scope_type = ?")
		args = append(args, scopeType)
	}
	if scopeID != "" {
		where = append(where, "scope_id = ?")
		args = append(args, scopeID)
	}

	query := fmt.Sprintf(`
		SELECT id, scope_type, scope_id, text, valid_from, valid_to,
			decay_policy, decayed_at, source, actor, created_at, expired
		FROM memos
		WHERE %s
		ORDER BY created_at DESC`, strings.Join(where, " AND "))

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list memos: %w", err)
	}
	defer rows.Close()

	return scanMemos(rows)
}

// Search 搜索 memo 文本。
func (s *MemoStore) Search(query string, scopeType, scopeID string, limit int) ([]model.Memo, error) {
	if limit <= 0 {
		limit = 10
	}

	where := []string{"1=1"}
	args := []interface{}{}

	if scopeType != "" {
		where = append(where, "scope_type = ?")
		args = append(args, scopeType)
	}
	if scopeID != "" {
		where = append(where, "scope_id = ?")
		args = append(args, scopeID)
	}

	whereClause := strings.Join(where, " AND ")

	// FTS 搜索
	searchArgs := append([]interface{}{query}, args...)
	searchArgs = append(searchArgs, limit)
	rows, err := s.db.Query(fmt.Sprintf(`
		SELECT m.id, m.scope_type, m.scope_id, m.text, m.valid_from, m.valid_to,
			m.decay_policy, m.decayed_at, m.source, m.actor, m.created_at, m.expired
		FROM memos m
		INNER JOIN memos_fts f ON m.id = f.id
		WHERE memos_fts MATCH ? AND %s
		ORDER BY rank
		LIMIT ?`, whereClause), searchArgs...)

	if err == nil {
		defer rows.Close()
		results, err2 := scanMemos(rows)
		if err2 == nil && len(results) > 0 {
			return results, nil
		}
	} else if rows != nil {
		rows.Close()
	}

	// fallback 到 LIKE（CJK 文本 FTS 可能返回空）
	likeArgs := append([]interface{}{"%" + query + "%"}, args...)
	likeArgs = append(likeArgs, limit)
	rows, err = s.db.Query(fmt.Sprintf(`
		SELECT id, scope_type, scope_id, text, valid_from, valid_to,
			decay_policy, decayed_at, source, actor, created_at, expired
		FROM memos
		WHERE text LIKE ? AND %s
		LIMIT ?`, whereClause), likeArgs...)
	if err != nil {
		return nil, fmt.Errorf("search memos: %w", err)
	}
	defer rows.Close()

	return scanMemos(rows)
}

// MarkExpired 标记一条 memo 为过期。
func (s *MemoStore) MarkExpired(id string) error {
	_, err := s.db.Exec(`UPDATE memos SET expired = 1, decayed_at = ? WHERE id = ?`, Now(), id)
	return err
}

// GetByID 通过 ID 获取 memo。
func (s *MemoStore) GetByID(id string) (*model.Memo, error) {
	row := s.db.QueryRow(`
		SELECT id, scope_type, scope_id, text, valid_from, valid_to,
			decay_policy, decayed_at, source, actor, created_at, expired
		FROM memos WHERE id = ?`, id)

	var m model.Memo
	err := row.Scan(&m.ID, &m.ScopeType, &m.ScopeID, &m.Text, &m.ValidFrom, &m.ValidTo,
		&m.Decay, &m.ExpiredAt, &m.Source, &m.Actor, &m.CreatedAt, &m.Expired)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("memo not found")
		}
		return nil, fmt.Errorf("get memo: %w", err)
	}
	return &m, nil
}

// Delete 删除 memo。
func (s *MemoStore) Delete(id string) error {
	_, err := s.db.Exec(`DELETE FROM memos WHERE id = ?`, id)
	return err
}

// TextSimilarity 计算两个文本的字符级 Jaccard 相似度（基于 bigram）。
func TextSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}
	if a == "" || b == "" {
		return 0.0
	}

	gramsA := make(map[string]bool)
	gramsB := make(map[string]bool)

	runesA := []rune(a)
	runesB := []rune(b)

	for i := 0; i < len(runesA)-1; i++ {
		gramsA[string(runesA[i:i+2])] = true
	}
	for i := 0; i < len(runesB)-1; i++ {
		gramsB[string(runesB[i:i+2])] = true
	}

	if len(gramsA) == 0 && len(gramsB) == 0 {
		return 1.0
	}

	intersection := 0
	for g := range gramsA {
		if gramsB[g] {
			intersection++
		}
	}
	union := len(gramsA) + len(gramsB) - intersection
	if union == 0 {
		return 0.0
	}
	return float64(intersection) / float64(union)
}

// Propose 提议一条新记忆，返回是否存在冲突。
func (s *MemoStore) Propose(scope, statement, sourceSnippet, actor string, confidence float64) (*model.Proposal, error) {
	parts := strings.SplitN(scope, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("无效 scope 格式，应如 contact:<id>")
	}
	scopeType, scopeID := parts[0], parts[1]

	p := &model.Proposal{
		ID:            model.NewID("prop"),
		Timestamp:     Now(),
		Actor:         actor,
		Scope:         scope,
		Statement:     statement,
		SourceSnippet: sourceSnippet,
		Confidence:    confidence,
		Status:        "clean",
	}

	existing, err := s.ListByScope(scopeType, scopeID)
	if err != nil {
		return p, nil
	}

	threshold := 0.7
	for _, m := range existing {
		if m.Expired {
			continue
		}
		sim := TextSimilarity(statement, m.Text)
		if sim >= threshold {
			p.Status = "conflict"
			p.ConflictWith = append(p.ConflictWith, model.ConflictItem{
				MemoID:    m.ID,
				Text:      m.Text,
				ValidFrom: m.ValidFrom,
			})
		}
	}

	if p.Status == "conflict" {
		p.SuggestedAction = "supersede"
	}

	return p, nil
}

// Commit 执行提议裁决。
func (s *MemoStore) Commit(proposalID, action string) error {
	proposals, err := s.readProposals()
	if err != nil {
		return fmt.Errorf("read proposals: %w", err)
	}

	var found *model.Proposal
	for i := range proposals {
		if proposals[i].ID == proposalID {
			found = &proposals[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("proposal not found: %s", proposalID)
	}

	switch action {
	case "supersede":
		for _, c := range found.ConflictWith {
			if c.MemoID != "" {
				_ = s.MarkExpired(c.MemoID)
			}
		}
		parts := strings.SplitN(found.Scope, ":", 2)
		if len(parts) == 2 {
			m := &model.Memo{
				ID:        model.MemoID(),
				ScopeType: parts[0],
				ScopeID:   parts[1],
				Text:      found.Statement,
				Source:    found.SourceSnippet,
				Actor:     found.Actor,
			}
			_ = s.Insert(m)
		}
	case "keep-both":
		parts := strings.SplitN(found.Scope, ":", 2)
		if len(parts) == 2 {
			m := &model.Memo{
				ID:        model.MemoID(),
				ScopeType: parts[0],
				ScopeID:   parts[1],
				Text:      found.Statement,
				Source:    found.SourceSnippet,
				Actor:     found.Actor,
			}
			_ = s.Insert(m)
		}
	case "reject":
	default:
		return fmt.Errorf("无效动作: %s (支持: supersede, keep-both, reject)", action)
	}

	return nil
}

// DecayScan 扫描并标记过期记忆。
func (s *MemoStore) DecayScan(dryRun bool) (expiredCount int, err error) {
	memos, err := s.ListByScope("", "")
	if err != nil {
		return 0, err
	}

	for _, m := range memos {
		if m.Expired {
			continue
		}
		if m.IsExpired("") {
			expiredCount++
			if !dryRun {
				_ = s.MarkExpired(m.ID)
			}
		}
	}

	return expiredCount, nil
}

func (s *MemoStore) readProposals() ([]model.Proposal, error) {
	path := filepath.Join(s.configDir, "proposals", "pending.jsonl")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var proposals []model.Proposal
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var p model.Proposal
		if err := json.Unmarshal(line, &p); err != nil {
			continue
		}
		proposals = append(proposals, p)
	}
	return proposals, scanner.Err()
}

func scanMemos(rows *sql.Rows) ([]model.Memo, error) {
	var memos []model.Memo
	for rows.Next() {
		var m model.Memo
		if err := rows.Scan(&m.ID, &m.ScopeType, &m.ScopeID, &m.Text, &m.ValidFrom, &m.ValidTo,
			&m.Decay, &m.ExpiredAt, &m.Source, &m.Actor, &m.CreatedAt, &m.Expired); err != nil {
			return nil, fmt.Errorf("scan memo: %w", err)
		}
		memos = append(memos, m)
	}
	return memos, nil
}
