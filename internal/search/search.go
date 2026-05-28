package search

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	weightFTS      = 0.5
	weightEntity   = 0.3
	weightTemporal = 0.2
)

// ContactResult 融合搜索结果，附带相关性分数。
type ContactResult struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Company        string  `json:"company,omitempty"`
	Title          string  `json:"title,omitempty"`
	Email          string  `json:"email,omitempty"`
	Tags           string  `json:"tags,omitempty"`
	LastActivityAt string  `json:"last_activity_at,omitempty"`
	Score          float64 `json:"score"`
}

// MemoResult 记忆搜索结果。
type MemoResult struct {
	ID        string  `json:"id"`
	ScopeType string  `json:"scope_type"`
	ScopeID   string  `json:"scope_id"`
	Text      string  `json:"text"`
	Source    string  `json:"source,omitempty"`
	Actor     string  `json:"actor,omitempty"`
	CreatedAt string  `json:"created_at,omitempty"`
	Expired   bool    `json:"expired"`
	Score     float64 `json:"score"`
}

// Engine 提供多策略融合检索。
type Engine struct {
	db *sql.DB
}

// NewEngine 创建检索引擎。
func NewEngine(db *sql.DB) *Engine {
	return &Engine{db: db}
}

// SearchContacts 执行融合搜索，返回按相关性排序的联系人。
func (e *Engine) SearchContacts(query string, limit int) ([]ContactResult, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := e.db.Query(`
		SELECT c.id, c.name, c.company, c.title, c.emails, c.last_activity_at, c.tags, c.updated_at, rank
		FROM contacts c
		INNER JOIN contacts_fts f ON c.id = f.id
		WHERE contacts_fts MATCH ?
		ORDER BY rank
		LIMIT ?`, query, limit*3)
	if err != nil {
		return e.searchContactLike(query, limit)
	}
	defer rows.Close()

	candidates, err := scanContactFTS(rows)
	if err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		return e.searchContactLike(query, limit)
	}

	// 计算 FTS 分数（rank 越负匹配越好）
	var maxRank float64
	for _, c := range candidates {
		if c.ftsRank < maxRank {
			maxRank = c.ftsRank
		}
	}
	for i, c := range candidates {
		ftsScore := ftsNormalize(c.ftsRank, maxRank)
		entityScore := entityContactScore(query, c.Name, c.Company, c.Email)
		temporalScore := temporalScore(c.LastActivityAt)
		candidates[i].Score = weightFTS*ftsScore + weightEntity*entityScore + weightTemporal*temporalScore
	}

	return finalizeContactResults(candidates, limit)
}

func (e *Engine) searchContactLike(query string, limit int) ([]ContactResult, error) {
	rows, err := e.db.Query(`
		SELECT id, name, company, title, emails, last_activity_at, tags, updated_at
		FROM contacts
		WHERE name LIKE ? OR company LIKE ? OR emails LIKE ?
		ORDER BY updated_at DESC
		LIMIT ?`,
		"%"+query+"%", "%"+query+"%", "%"+query+"%", limit*3)
	if err != nil {
		return nil, fmt.Errorf("search contacts: %w", err)
	}
	defer rows.Close()

	candidates, err := scanContactCandidates(rows)
	if err != nil {
		return nil, err
	}

	for i, c := range candidates {
		entityScore := entityContactScore(query, c.Name, c.Company, c.Email)
		temporalScore := temporalScore(c.LastActivityAt)
		candidates[i].Score = weightEntity*entityScore + weightTemporal*temporalScore
	}

	return finalizeContactResults(candidates, limit)
}

// SearchMemos 执行记忆融合搜索。
func (e *Engine) SearchMemos(query, scopeType, scopeID string, limit int) ([]MemoResult, error) {
	if limit <= 0 {
		limit = 10
	}

	where := []string{"1=1"}
	args := []interface{}{}

	if scopeType != "" {
		where = append(where, "m.scope_type = ?")
		args = append(args, scopeType)
	}
	if scopeID != "" {
		where = append(where, "m.scope_id = ?")
		args = append(args, scopeID)
	}
	whereClause := strings.Join(where, " AND ")

	ftsArgs := append([]interface{}{query}, args...)
	ftsArgs = append(ftsArgs, limit*3)

	rows, err := e.db.Query(fmt.Sprintf(`
		SELECT m.id, m.scope_type, m.scope_id, m.text, m.source, m.actor, m.created_at, m.expired, rank
		FROM memos m
		INNER JOIN memos_fts f ON m.id = f.id
		WHERE memos_fts MATCH ? AND %s
		ORDER BY rank
		LIMIT ?`, whereClause), ftsArgs...)
	if err != nil {
		return e.searchMemoLike(query, scopeType, scopeID, limit)
	}
	defer rows.Close()

	candidates, err := scanMemoFTS(rows)
	if err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		return e.searchMemoLike(query, scopeType, scopeID, limit)
	}

	var maxRank float64
	for _, c := range candidates {
		if c.ftsRank < maxRank {
			maxRank = c.ftsRank
		}
	}
	for i, c := range candidates {
		ftsScore := ftsNormalize(c.ftsRank, maxRank)
		entityScore := entityMemoScore(query, c.Text)
		temporalScore := temporalScore(c.CreatedAt)
		candidates[i].Score = weightFTS*ftsScore + weightEntity*entityScore + weightTemporal*temporalScore
	}

	return finalizeMemoResults(candidates, limit)
}

func (e *Engine) searchMemoLike(query, scopeType, scopeID string, limit int) ([]MemoResult, error) {
	where := []string{"text LIKE ?"}
	args := []interface{}{"%" + query + "%"}

	if scopeType != "" {
		where = append(where, "scope_type = ?")
		args = append(args, scopeType)
	}
	if scopeID != "" {
		where = append(where, "scope_id = ?")
		args = append(args, scopeID)
	}
	args = append(args, limit*3)

	rows, err := e.db.Query(fmt.Sprintf(`
		SELECT id, scope_type, scope_id, text, source, actor, created_at, expired
		FROM memos
		WHERE %s
		ORDER BY created_at DESC
		LIMIT ?`, strings.Join(where, " AND ")), args...)
	if err != nil {
		return nil, fmt.Errorf("search memos: %w", err)
	}
	defer rows.Close()

	candidates, err := scanMemoCandidates(rows)
	if err != nil {
		return nil, err
	}

	for i, c := range candidates {
		entityScore := entityMemoScore(query, c.Text)
		temporalScore := temporalScore(c.CreatedAt)
		candidates[i].Score = weightEntity*entityScore + weightTemporal*temporalScore
	}

	return finalizeMemoResults(candidates, limit)
}

// --- internal types for scoring ---

type contactCandidate struct {
	ContactResult
	ftsRank   float64
}

type memoCandidate struct {
	MemoResult
	ftsRank   float64
}

// --- scoring functions ---

func ftsNormalize(rank, maxRank float64) float64 {
	if maxRank >= 0 {
		return 0.5
	}
	// rank is negative (FTS5 BM25), lower = better
	// normalize relative to the best rank in the set
	rel := rank / maxRank // worst = 0, best = 1
	if rel > 1 {
		rel = 1
	}
	return rel
}

func entityContactScore(query, name, company, email string) float64 {
	q := strings.ToLower(strings.TrimSpace(query))
	n := strings.ToLower(name)
	c := strings.ToLower(company)
	e := strings.ToLower(email)

	if n == q {
		return 1.0
	}
	if strings.HasPrefix(n, q) {
		return 0.8
	}
	// word-level prefix match
	for _, word := range strings.Fields(n) {
		if strings.HasPrefix(word, q) {
			return 0.75
		}
	}
	if strings.Contains(n, q) {
		return 0.6
	}
	if c != "" && strings.HasPrefix(c, q) {
		return 0.55
	}
	if c != "" && strings.Contains(c, q) {
		return 0.5
	}
	if e != "" && strings.HasPrefix(e, q) {
		return 0.4
	}
	return 0.0
}

func entityMemoScore(query, text string) float64 {
	q := strings.ToLower(strings.TrimSpace(query))
	t := strings.ToLower(text)

	if t == q {
		return 1.0
	}
	if strings.HasPrefix(t, q) {
		return 0.8
	}
	if strings.Contains(t, q) {
		return 0.6
	}
	return 0.0
}

func temporalScore(ts string) float64 {
	if ts == "" {
		return 0.3
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		t, err = time.Parse("2006-01-02", ts)
		if err != nil {
			return 0.3
		}
	}
	days := time.Since(t).Hours() / 24
	if days < 0 {
		days = 0
	}
	return math.Max(0, 1.0-days/365)
}

// --- scan helpers ---

func scanContactCandidates(rows *sql.Rows) ([]contactCandidate, error) {
	var candidates []contactCandidate
	for rows.Next() {
		var c contactCandidate
		var emailsJSON, tagsJSON, updatedAt string
		if err := rows.Scan(&c.ID, &c.Name, &c.Company, &c.Title,
			&emailsJSON, &c.LastActivityAt, &tagsJSON, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		var emails []string
		json.Unmarshal([]byte(emailsJSON), &emails)
		if len(emails) > 0 {
			c.Email = emails[0]
		}
		var tags []string
		json.Unmarshal([]byte(tagsJSON), &tags)
		c.Tags = strings.Join(tags, ", ")

		// for temporal scoring: prefer last_activity_at, fall back to updated_at
		if c.LastActivityAt == "" {
			c.LastActivityAt = updatedAt
		}

		candidates = append(candidates, c)
	}
	return candidates, nil
}

func scanMemoCandidates(rows *sql.Rows) ([]memoCandidate, error) {
	var candidates []memoCandidate
	for rows.Next() {
		var c memoCandidate
		var expiredInt int
		if err := rows.Scan(&c.ID, &c.ScopeType, &c.ScopeID, &c.Text,
			&c.Source, &c.Actor, &c.CreatedAt, &expiredInt); err != nil {
			return nil, fmt.Errorf("scan memo: %w", err)
		}
		c.Expired = expiredInt != 0
		candidates = append(candidates, c)
	}
	return candidates, nil
}

// note: scanContactCandidates and scanMemoCandidates above don't scan ftsRank
// because the LIKE fallback has no rank column. The FTS query must include rank.
// We handle this by having separate scan paths.

func scanContactFTS(rows *sql.Rows) ([]contactCandidate, error) {
	var candidates []contactCandidate
	for rows.Next() {
		var c contactCandidate
		var emailsJSON, tagsJSON, updatedAt string
		if err := rows.Scan(&c.ID, &c.Name, &c.Company, &c.Title,
			&emailsJSON, &c.LastActivityAt, &tagsJSON, &updatedAt, &c.ftsRank); err != nil {
			return nil, fmt.Errorf("scan fts: %w", err)
		}
		var emails []string
		json.Unmarshal([]byte(emailsJSON), &emails)
		if len(emails) > 0 {
			c.Email = emails[0]
		}
		var tags []string
		json.Unmarshal([]byte(tagsJSON), &tags)
		c.Tags = strings.Join(tags, ", ")

		if c.LastActivityAt == "" {
			c.LastActivityAt = updatedAt
		}

		candidates = append(candidates, c)
	}
	return candidates, nil
}

func scanMemoFTS(rows *sql.Rows) ([]memoCandidate, error) {
	var candidates []memoCandidate
	for rows.Next() {
		var c memoCandidate
		var expiredInt int
		if err := rows.Scan(&c.ID, &c.ScopeType, &c.ScopeID, &c.Text,
			&c.Source, &c.Actor, &c.CreatedAt, &expiredInt, &c.ftsRank); err != nil {
			return nil, fmt.Errorf("scan memo fts: %w", err)
		}
		c.Expired = expiredInt != 0
		candidates = append(candidates, c)
	}
	return candidates, nil
}

// --- finalize helpers ---

func finalizeContactResults(candidates []contactCandidate, limit int) ([]ContactResult, error) {
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	results := make([]ContactResult, len(candidates))
	for i, c := range candidates {
		results[i] = c.ContactResult
	}
	return results, nil
}

func finalizeMemoResults(candidates []memoCandidate, limit int) ([]MemoResult, error) {
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	results := make([]MemoResult, len(candidates))
	for i, c := range candidates {
		results[i] = c.MemoResult
	}
	return results, nil
}
