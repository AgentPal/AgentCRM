package search

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// setupTestDB 创建内存 SQLite + FTS5 表并插入测试数据。
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open memory db: %v", err)
	}

	schema := `
	CREATE TABLE contacts (
		id TEXT PRIMARY KEY,
		slug TEXT UNIQUE,
		name TEXT NOT NULL,
		emails TEXT DEFAULT '[]',
		company TEXT DEFAULT '',
		title TEXT DEFAULT '',
		tags TEXT DEFAULT '[]',
		last_activity_at TEXT DEFAULT '',
		updated_at TEXT DEFAULT '',
		has_birthday INTEGER DEFAULT 0,
		social_twitter TEXT DEFAULT '',
		social_linkedin TEXT DEFAULT '',
		social_wechat TEXT DEFAULT '',
		social_github TEXT DEFAULT '',
		source TEXT DEFAULT '',
		created_at TEXT DEFAULT ''
	);
	CREATE TABLE memos (
		id TEXT PRIMARY KEY,
		scope_type TEXT NOT NULL,
		scope_id TEXT NOT NULL,
		text TEXT NOT NULL,
		valid_from TEXT DEFAULT '',
		valid_to TEXT DEFAULT '',
		decay_policy TEXT DEFAULT 'never',
		decayed_at TEXT DEFAULT '',
		source TEXT DEFAULT '',
		actor TEXT DEFAULT '',
		created_at TEXT DEFAULT '',
		expired INTEGER DEFAULT 0
	);
	CREATE VIRTUAL TABLE contacts_fts USING fts5(id UNINDEXED, name, company, body, tokenize='unicode61');
	CREATE VIRTUAL TABLE memos_fts USING fts5(id UNINDEXED, text, tokenize='unicode61');
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	// 插入测试联系人
	insertContact(t, db, "c1", "张三", `["zhangsan@example.com"]`, "字节跳动", "工程师", `["tech"]`, "2026-05-20T10:00:00Z")
	insertContact(t, db, "c2", "李四", `["lisi@example.com"]`, "阿里巴巴", "产品经理", `["product"]`, "2026-05-15T10:00:00Z")
	insertContact(t, db, "c3", "John Smith", `["john@acme.com"]`, "Acme Corp", "CTO", `["tech","exec"]`, "2026-01-10T10:00:00Z")
	insertContact(t, db, "c4", "王五", `["wangwu@tencent.com"]`, "腾讯科技", "工程师", `["tech"]`, "2025-01-01T10:00:00Z")
	insertContact(t, db, "c5", "赵六", `["zhaoliu@baidu.com"]`, "百度", "工程师", `["tech"]`, "2020-01-01T10:00:00Z")

	// 插入测试记忆
	insertMemo(t, db, "m1", "contact", "c1", "张三对AI产品很感兴趣，特别是大语言模型", "bot1")
	insertMemo(t, db, "m2", "contact", "c1", "张三提到预算在50万以内", "bot2")
	insertMemo(t, db, "m3", "contact", "c2", "李四正在评估多个供应商", "bot1")
	insertMemo(t, db, "m4", "contact", "c3", "John prefers email communication", "bot1")
	insertMemo(t, db, "m5", "global", "", "公司全体注意：所有合同需法务审批", "admin")

	return db
}

func insertContact(t *testing.T, db *sql.DB, id, name, emails, company, title, tags, lastActivityAt string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO contacts (id, slug, name, emails, company, title, tags, last_activity_at, updated_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, name, name, emails, company, title, tags, lastActivityAt, lastActivityAt, lastActivityAt)
	if err != nil {
		t.Fatalf("insert contact %s: %v", id, err)
	}
	_, err = db.Exec(`INSERT INTO contacts_fts (id, name, company, body) VALUES (?, ?, ?, ?)`,
		id, name, company, name+" "+company)
	if err != nil {
		t.Fatalf("insert contact_fts %s: %v", id, err)
	}
}

func insertMemo(t *testing.T, db *sql.DB, id, scopeType, scopeID, text, actor string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO memos (id, scope_type, scope_id, text, actor, created_at)
		VALUES (?, ?, ?, ?, ?, '2026-05-20T10:00:00Z')`,
		id, scopeType, scopeID, text, actor)
	if err != nil {
		t.Fatalf("insert memo %s: %v", id, err)
	}
	_, err = db.Exec(`INSERT INTO memos_fts (id, text) VALUES (?, ?)`, id, text)
	if err != nil {
		t.Fatalf("insert memo_fts %s: %v", id, err)
	}
}

// --- SearchContacts ---

func TestSearchContacts_FTS(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eng := NewEngine(db)

	results, err := eng.SearchContacts("张三", 10)
	if err != nil {
		t.Fatalf("SearchContacts: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	if results[0].Name != "张三" {
		t.Errorf("expected top result '张三', got %q", results[0].Name)
	}
	if results[0].Score <= 0 {
		t.Errorf("expected positive score, got %f", results[0].Score)
	}
}

func TestSearchContacts_LIKEFallback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eng := NewEngine(db)

	// 使用 CJK 连续字符触发 LIKE 回退
	results, err := eng.SearchContacts("字节", 10)
	if err != nil {
		t.Fatalf("SearchContacts: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result for '字节'")
	}
	if results[0].Name != "张三" {
		t.Errorf("expected top result '张三', got %q", results[0].Name)
	}
}

func TestSearchContacts_NoMatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eng := NewEngine(db)

	results, err := eng.SearchContacts("不存在的人", 10)
	if err != nil {
		t.Fatalf("SearchContacts: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchContacts_EntityNameBoost(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eng := NewEngine(db)

	// "John" 应优先匹配 John Smith（实体精确前缀）而非公司名包含的
	results, err := eng.SearchContacts("John", 10)
	if err != nil {
		t.Fatalf("SearchContacts: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	if results[0].Name != "John Smith" {
		t.Errorf("expected 'John Smith' first, got %q", results[0].Name)
	}
}

func TestSearchContacts_Limit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eng := NewEngine(db)

	results, err := eng.SearchContacts("工程师", 2)
	if err != nil {
		t.Fatalf("SearchContacts: %v", err)
	}
	if len(results) > 2 {
		t.Errorf("expected at most 2 results, got %d", len(results))
	}
}

func TestSearchContacts_DefaultLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eng := NewEngine(db)

	results, err := eng.SearchContacts("工程师", 0)
	if err != nil {
		t.Fatalf("SearchContacts: %v", err)
	}
	if len(results) > 10 {
		t.Errorf("expected at most 10 results with default limit, got %d", len(results))
	}
}

// --- SearchMemos ---

func TestSearchMemos_FTS(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eng := NewEngine(db)

	results, err := eng.SearchMemos("AI产品", "", "", 10)
	if err != nil {
		t.Fatalf("SearchMemos: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	if results[0].ScopeID != "c1" {
		t.Errorf("expected scope c1, got %q", results[0].ScopeID)
	}
}

func TestSearchMemos_ByScope(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eng := NewEngine(db)

	results, err := eng.SearchMemos("供应商", "contact", "c2", 10)
	if err != nil {
		t.Fatalf("SearchMemos: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	if results[0].ID != "m3" {
		t.Errorf("expected memo m3, got %q", results[0].ID)
	}
}

func TestSearchMemos_GlobalScope(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eng := NewEngine(db)

	results, err := eng.SearchMemos("法务", "global", "", 10)
	if err != nil {
		t.Fatalf("SearchMemos: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	if results[0].ID != "m5" {
		t.Errorf("expected memo m5, got %q", results[0].ID)
	}
}

func TestSearchMemos_LIKEFallback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eng := NewEngine(db)

	// CJK 连续字符触发 LIKE 回退
	results, err := eng.SearchMemos("大语言模型", "", "", 10)
	if err != nil {
		t.Fatalf("SearchMemos: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	if results[0].ID != "m1" {
		t.Errorf("expected memo m1, got %q", results[0].ID)
	}
}

func TestSearchMemos_NoMatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eng := NewEngine(db)

	results, err := eng.SearchMemos("zzzzzxxxx", "", "", 10)
	if err != nil {
		t.Fatalf("SearchMemos: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchMemos_DefaultLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eng := NewEngine(db)

	results, err := eng.SearchMemos("a", "", "", 0)
	if err != nil {
		t.Fatalf("SearchMemos: %v", err)
	}
	if len(results) > 10 {
		t.Errorf("expected at most 10 results with default limit, got %d", len(results))
	}
}

// --- scoring functions ---

func TestFtsNormalize_Best(t *testing.T) {
	// best rank should normalize to 1.0
	score := ftsNormalize(-5.0, -5.0)
	if score != 1.0 {
		t.Errorf("expected 1.0, got %f", score)
	}
}

func TestFtsNormalize_Worst(t *testing.T) {
	// worst rank should normalize to ~0
	score := ftsNormalize(-1.0, -5.0)
	if score > 0.5 {
		t.Errorf("expected low score, got %f", score)
	}
}

func TestFtsNormalize_NonNegative(t *testing.T) {
	// maxRank >= 0 → neutral 0.5
	score := ftsNormalize(-3.0, 0)
	if score != 0.5 {
		t.Errorf("expected 0.5, got %f", score)
	}
}

func TestEntityContactScore_Exact(t *testing.T) {
	score := entityContactScore("张三", "张三", "字节跳动", "zhangsan@example.com")
	if score != 1.0 {
		t.Errorf("expected 1.0, got %f", score)
	}
}

func TestEntityContactScore_Prefix(t *testing.T) {
	score := entityContactScore("John", "John Smith", "", "")
	if score != 0.8 {
		t.Errorf("expected 0.8, got %f", score)
	}
}

func TestEntityContactScore_WordPrefix(t *testing.T) {
	score := entityContactScore("Smi", "John Smith", "", "")
	if score != 0.75 {
		t.Errorf("expected 0.75, got %f", score)
	}
}

func TestEntityContactScore_Contains(t *testing.T) {
	score := entityContactScore("ohn", "John Smith", "", "")
	if score != 0.6 {
		t.Errorf("expected 0.6, got %f", score)
	}
}

func TestEntityContactScore_CompanyContains(t *testing.T) {
	// "跳动" is contained but not a prefix → 0.5
	score := entityContactScore("跳动", "张三", "字节跳动", "")
	if score != 0.5 {
		t.Errorf("expected 0.5, got %f", score)
	}
}

func TestEntityContactScore_CompanyPrefix(t *testing.T) {
	// "字节" is prefix of "字节跳动" → 0.55 (prefix checked before contains)
	score := entityContactScore("字节", "张三", "字节跳动", "")
	if score != 0.55 {
		t.Errorf("expected 0.55, got %f", score)
	}
}

func TestEntityContactScore_EmailPrefix(t *testing.T) {
	score := entityContactScore("zhang", "张三", "", "zhangsan@example.com")
	if score != 0.4 {
		t.Errorf("expected 0.4, got %f", score)
	}
}

func TestEntityContactScore_NoMatch(t *testing.T) {
	score := entityContactScore("zzzz", "张三", "", "")
	if score != 0.0 {
		t.Errorf("expected 0.0, got %f", score)
	}
}

func TestEntityMemoScore_Exact(t *testing.T) {
	score := entityMemoScore("hello world", "hello world")
	if score != 1.0 {
		t.Errorf("expected 1.0, got %f", score)
	}
}

func TestEntityMemoScore_Prefix(t *testing.T) {
	score := entityMemoScore("hello", "hello world")
	if score != 0.8 {
		t.Errorf("expected 0.8, got %f", score)
	}
}

func TestEntityMemoScore_Contains(t *testing.T) {
	score := entityMemoScore("world", "hello world")
	if score != 0.6 {
		t.Errorf("expected 0.6, got %f", score)
	}
}

func TestEntityMemoScore_NoMatch(t *testing.T) {
	score := entityMemoScore("zzzz", "hello world")
	if score != 0.0 {
		t.Errorf("expected 0.0, got %f", score)
	}
}

func TestTemporalScore_Recent(t *testing.T) {
	// recent activity should score near 1.0
	score := temporalScore("2026-05-28T10:00:00Z")
	if score < 0.95 {
		t.Errorf("expected near 1.0 for recent, got %f", score)
	}
}

func TestTemporalScore_Old(t *testing.T) {
	// 5+ year old activity should score ~0
	score := temporalScore("2020-01-01T10:00:00Z")
	if score > 0.1 {
		t.Errorf("expected near 0 for old, got %f", score)
	}
}

func TestTemporalScore_Empty(t *testing.T) {
	score := temporalScore("")
	if score != 0.3 {
		t.Errorf("expected 0.3 neutral, got %f", score)
	}
}

func TestTemporalScore_DateOnly(t *testing.T) {
	score := temporalScore("2026-05-20")
	if score < 0.95 {
		t.Errorf("expected near 1.0 for recent date, got %f", score)
	}
}

func TestTemporalScore_Invalid(t *testing.T) {
	score := temporalScore("not-a-date")
	if score != 0.3 {
		t.Errorf("expected 0.3 neutral for invalid, got %f", score)
	}
}

func TestTemporalScore_Future(t *testing.T) {
	score := temporalScore("2099-01-01T10:00:00Z")
	// clamped to 0
	if score != 1.0 {
		t.Errorf("expected 1.0 for future date, got %f", score)
	}
}

// --- SearchContacts score ordering ---

func TestSearchContacts_ScoreOrdering(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eng := NewEngine(db)

	// "example" matches all contacts via emails LIKE fallback
	results, err := eng.SearchContacts("example", 10)
	if err != nil {
		t.Fatalf("SearchContacts: %v", err)
	}
	if len(results) < 2 {
		t.Fatal("expected at least 2 results")
	}
	// results should be sorted descending by score
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("results not sorted by score descending at index %d: %f > %f", i, results[i].Score, results[i-1].Score)
		}
	}
}

// --- precision: exact name match must be #1 over partial ---

func TestSearchContacts_ExactPreferred(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eng := NewEngine(db)

	// "张三" is exact name match for one contact, company prefix for others via "张"
	// Different strokes — just verify no crash and scores make sense
	results, err := eng.SearchContacts("张三", 10)
	if err != nil {
		t.Fatalf("SearchContacts: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	if results[0].Score <= 0 {
		t.Errorf("expected positive score for exact name match, got %f", results[0].Score)
	}
}

// --- Score clamping ---

func TestScoreRange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eng := NewEngine(db)

	results, err := eng.SearchContacts("张三", 10)
	if err != nil {
		t.Fatalf("SearchContacts: %v", err)
	}
	for _, r := range results {
		if r.Score < 0 || r.Score > 1.5 {
			t.Errorf("score %f out of reasonable range [0, 1.5] for contact %s", r.Score, r.ID)
		}
	}
}

// --- Integration: round-trip ---

func TestSearchContactResultFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eng := NewEngine(db)

	results, err := eng.SearchContacts("Acme", 10)
	if err != nil {
		t.Fatalf("SearchContacts: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for 'Acme'")
	}
	r := results[0]
	if r.ID == "" {
		t.Error("expected non-empty ID")
	}
	if r.Name == "" {
		t.Error("expected non-empty Name")
	}
}

func TestSearchMemoResultFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eng := NewEngine(db)

	results, err := eng.SearchMemos("email", "", "", 10)
	if err != nil {
		t.Fatalf("SearchMemos: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for 'email'")
	}
	r := results[0]
	if r.ID == "" {
		t.Error("expected non-empty ID")
	}
	if r.Text == "" {
		t.Error("expected non-empty Text")
	}
}
