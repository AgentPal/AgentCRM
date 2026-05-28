package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AgentPal/AgentCRM/internal/model"
)

// DealStore 提供对商机索引的数据库操作。
type DealStore struct {
	db *DB
}

func NewDealStore(db *DB) *DealStore {
	return &DealStore{db: db}
}

// Create 创建一个商机。
func (s *DealStore) Create(d *model.Deal) error {
	contactIDsJSON, _ := json.Marshal(d.ContactIDs)
	_, err := s.db.Exec(`
		INSERT INTO deals (id, slug, title, stage, amount, currency,
			contact_ids, expected_close_at, owner, updated_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ID, d.Slug, d.Title, d.Stage, d.Amount, d.Currency,
		string(contactIDsJSON), d.ExpectedCloseAt, d.Owner, d.UpdatedAt, d.CreatedAt)
	return err
}

// GetByID 通过 ID 获取商机。
func (s *DealStore) GetByID(id string) (*model.Deal, error) {
	var d model.Deal
	var contactIDsJSON string
	err := s.db.QueryRow(`
		SELECT id, slug, title, stage, amount, currency,
			contact_ids, expected_close_at, owner, updated_at, created_at
		FROM deals WHERE id = ?`, id).Scan(
		&d.ID, &d.Slug, &d.Title, &d.Stage, &d.Amount, &d.Currency,
		&contactIDsJSON, &d.ExpectedCloseAt, &d.Owner, &d.UpdatedAt, &d.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("deal not found")
		}
		return nil, fmt.Errorf("get deal: %w", err)
	}
	json.Unmarshal([]byte(contactIDsJSON), &d.ContactIDs)
	return &d, nil
}

// List 列出符合条件的商机。
func (s *DealStore) List(stage, stageNotIn, owner string) ([]model.DealSearchResult, error) {
	where := []string{"1=1"}
	args := []interface{}{}

	if stage != "" {
		where = append(where, "stage = ?")
		args = append(args, stage)
	}
	if stageNotIn != "" {
		stages := strings.Split(stageNotIn, ",")
		placeholders := make([]string, len(stages))
		for i, st := range stages {
			placeholders[i] = "?"
			args = append(args, strings.TrimSpace(st))
		}
		where = append(where, fmt.Sprintf("stage NOT IN (%s)", strings.Join(placeholders, ",")))
	}
	if owner != "" {
		where = append(where, "owner = ?")
		args = append(args, owner)
	}

	query := fmt.Sprintf(`
		SELECT d.id, d.title, d.stage, d.amount, d.currency,
			d.expected_close_at, d.updated_at, d.contact_ids
		FROM deals d
		WHERE %s
		ORDER BY d.updated_at DESC
		LIMIT 100`, strings.Join(where, " AND "))

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list deals: %w", err)
	}
	defer rows.Close()

	var results []model.DealSearchResult
	for rows.Next() {
		var r model.DealSearchResult
		var contactIDsJSON string
		if err := rows.Scan(&r.ID, &r.Title, &r.Stage, &r.Amount, &r.Currency,
			&r.ExpectedCloseAt, &r.UpdatedAt, &contactIDsJSON); err != nil {
			return nil, fmt.Errorf("scan deal: %w", err)
		}
		results = append(results, r)
	}
	return results, nil
}

// UpdateStage 更新阶段并返回旧值。
func (s *DealStore) UpdateStage(id, newStage, reason, actor string) (oldStage string, err error) {
	err = s.db.QueryRow(`SELECT stage FROM deals WHERE id = ?`, id).Scan(&oldStage)
	if err != nil {
		return "", fmt.Errorf("get current stage: %w", err)
	}

	if oldStage == newStage {
		return oldStage, nil
	}

	if !model.StageTransitionAllowed(oldStage, newStage) {
		return "", fmt.Errorf("cannot transition from %s to %s: won/lost is irreversible", oldStage, newStage)
	}

	_, err = s.db.Exec(`UPDATE deals SET stage = ?, updated_at = ? WHERE id = ?`,
		newStage, Now(), id)
	return oldStage, err
}

// UpdateField 更新商机字段。
func (s *DealStore) UpdateField(id, field string, value interface{}) error {
	_, err := s.db.Exec(fmt.Sprintf(`UPDATE deals SET %s = ?, updated_at = ? WHERE id = ?`,
		field), value, Now(), id)
	return err
}

// Delete 删除商机索引。
func (s *DealStore) Delete(id string) error {
	_, err := s.db.Exec(`DELETE FROM deals WHERE id = ?`, id)
	return err
}
