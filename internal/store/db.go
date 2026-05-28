package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// DB 封装 SQLite 连接，提供索引层的读写操作。
type DB struct {
	*sql.DB
}

// OpenDB 打开或创建 SQLite 数据库。
func OpenDB(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite 不支持并发写

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	wrapper := &DB{db}
	if err := wrapper.migrate(); err != nil {
		return nil, fmt.Errorf("migrate db: %w", err)
	}

	return wrapper, nil
}

// migrate 执行数据库建表/迁移。
func (d *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS contacts (
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

	CREATE TABLE IF NOT EXISTS deals (
		id TEXT PRIMARY KEY,
		slug TEXT UNIQUE,
		title TEXT NOT NULL,
		stage TEXT NOT NULL DEFAULT 'lead',
		amount INTEGER DEFAULT 0,
		currency TEXT DEFAULT 'CNY',
		contact_ids TEXT DEFAULT '[]',
		expected_close_at TEXT DEFAULT '',
		owner TEXT DEFAULT '',
		updated_at TEXT DEFAULT '',
		created_at TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS activities (
		id TEXT PRIMARY KEY,
		ts TEXT NOT NULL,
		type TEXT NOT NULL,
		channel TEXT DEFAULT '',
		direction TEXT DEFAULT '',
		actor TEXT DEFAULT '',
		contact_ids TEXT DEFAULT '[]',
		deal_ids TEXT DEFAULT '[]',
		summary TEXT DEFAULT '',
		body TEXT DEFAULT '',
		subject TEXT DEFAULT '',
		dedupe_key TEXT UNIQUE,
		source_url TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS events (
		seq INTEGER PRIMARY KEY AUTOINCREMENT,
		ts TEXT NOT NULL,
		actor TEXT NOT NULL,
		type TEXT NOT NULL,
		payload TEXT DEFAULT '{}'
	);

	CREATE TABLE IF NOT EXISTS memos (
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

	CREATE INDEX IF NOT EXISTS idx_memos_scope ON memos(scope_type, scope_id);
	CREATE INDEX IF NOT EXISTS idx_activities_contact ON activities(contact_ids);
	CREATE INDEX IF NOT EXISTS idx_activities_ts ON activities(ts);
	CREATE INDEX IF NOT EXISTS idx_deals_stage ON deals(stage);
	CREATE INDEX IF NOT EXISTS idx_events_seq ON events(seq);

	CREATE VIRTUAL TABLE IF NOT EXISTS contacts_fts USING fts5(
		id UNINDEXED, name, company, body,
		tokenize='unicode61'
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS activities_fts USING fts5(
		id UNINDEXED, summary, body,
		tokenize='unicode61'
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS memos_fts USING fts5(
		id UNINDEXED, text,
		tokenize='unicode61'
	);
	`
	_, err := d.Exec(schema)
	return err
}

// Now 返回当前 UTC 时间戳。
func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}
