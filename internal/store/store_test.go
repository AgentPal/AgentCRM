package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AgentPal/AgentCRM/internal/model"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return db
}

func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "agentcrm-test-*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// --- ContactStore ---

func TestContactUpsert_Create(t *testing.T) {
	db := openTestDB(t)
	s := NewContactStore(db)

	c := model.NewContact()
	c.ID = model.ContactID()
	c.Name = "张三"
	c.Emails = []string{"zhang@test.com"}
	c.Company = "测试公司"

	created, err := s.Upsert(c)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if !created {
		t.Error("expected created=true for new contact")
	}

	got, err := s.GetByID(c.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "张三" {
		t.Errorf("expected name '张三', got %q", got.Name)
	}
	if got.Company != "测试公司" {
		t.Errorf("expected company '测试公司', got %q", got.Company)
	}
}

func TestContactUpsert_UpdateByEmail(t *testing.T) {
	db := openTestDB(t)
	s := NewContactStore(db)

	c1 := model.NewContact()
	c1.ID = model.ContactID()
	c1.Name = "张三"
	c1.Emails = []string{"same@test.com"}
	s.Upsert(c1)

	c2 := model.NewContact()
	c2.ID = model.ContactID()
	c2.Name = "张三新名"
	c2.Emails = []string{"same@test.com"}
	c2.Company = "新公司"

	created, err := s.Upsert(c2)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if created {
		t.Error("expected created=false for update by email")
	}
	if c2.ID != c1.ID {
		t.Errorf("expected same ID after email dedup, got %q vs %q", c2.ID, c1.ID)
	}

	got, _ := s.GetByID(c1.ID)
	if got.Name != "张三新名" {
		t.Errorf("expected updated name '张三新名', got %q", got.Name)
	}
}

func TestContactGetByEmail(t *testing.T) {
	db := openTestDB(t)
	s := NewContactStore(db)

	c := model.NewContact()
	c.ID = model.ContactID()
	c.Name = "李四"
	c.Emails = []string{"lisi@test.com"}
	s.Upsert(c)

	got, err := s.GetByEmail("lisi@test.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if got.Name != "李四" {
		t.Errorf("expected '李四', got %q", got.Name)
	}
}

func TestContactGetByEmail_NotFound(t *testing.T) {
	db := openTestDB(t)
	s := NewContactStore(db)

	_, err := s.GetByEmail("nonexistent@test.com")
	if err == nil {
		t.Fatal("expected error for nonexistent email")
	}
}

func TestContactSearch_LIKE(t *testing.T) {
	db := openTestDB(t)
	s := NewContactStore(db)

	c := model.NewContact()
	c.ID = model.ContactID()
	c.Name = "王五"
	c.Emails = []string{"wang@test.com"}
	c.Company = "数据科技"
	s.Upsert(c)

	results, err := s.Search("王五", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results")
	}
	if results[0].Name != "王五" {
		t.Errorf("expected '王五', got %q", results[0].Name)
	}

	results, err = s.Search("数据", 10)
	if err != nil {
		t.Fatalf("Search company: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results for company query")
	}
}

func TestContactList_ByTag(t *testing.T) {
	db := openTestDB(t)
	s := NewContactStore(db)

	c := model.NewContact()
	c.ID = model.ContactID()
	c.Name = "VIP客户"
	c.Emails = []string{"vip@test.com"}
	c.Tags = []string{"vip", "partner"}
	s.Upsert(c)

	results, err := s.List("vip", "", "", 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected vip contact in list")
	}
	if !strings.Contains(results[0].Tags, "vip") {
		t.Errorf("expected vip tag, got %q", results[0].Tags)
	}
}

func TestContactUpdate(t *testing.T) {
	db := openTestDB(t)
	s := NewContactStore(db)

	c := model.NewContact()
	c.ID = model.ContactID()
	c.Name = "测试"
	c.Emails = []string{"test@test.com"}
	s.Upsert(c)

	oldVal, err := s.Update(c.ID, "company", "新公司", "测试")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if oldVal != "" {
		t.Errorf("expected empty old value, got %q", oldVal)
	}

	got, _ := s.GetByID(c.ID)
	if got.Company != "新公司" {
		t.Errorf("expected '新公司', got %q", got.Company)
	}
}

func TestContactDelete(t *testing.T) {
	db := openTestDB(t)
	s := NewContactStore(db)

	c := model.NewContact()
	c.ID = model.ContactID()
	c.Name = "待删除"
	c.Emails = []string{"del@test.com"}
	s.Upsert(c)

	err := s.Delete(c.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = s.GetByID(c.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

// --- DealStore ---

func TestDealCreate(t *testing.T) {
	db := openTestDB(t)
	s := NewDealStore(db)

	d := model.NewDeal()
	d.ID = model.DealID()
	d.Slug = "test-deal"
	d.Title = "测试商机"
	d.Amount = 50000
	d.Stage = model.StageQualified

	err := s.Create(d)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := s.GetByID(d.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Title != "测试商机" {
		t.Errorf("expected '测试商机', got %q", got.Title)
	}
	if got.Amount != 50000 {
		t.Errorf("expected amount 50000, got %d", got.Amount)
	}
}

func TestDealList_ByStage(t *testing.T) {
	db := openTestDB(t)
	s := NewDealStore(db)

	d1 := model.NewDeal()
	d1.ID = model.DealID()
	d1.Slug = "deal-1"
	d1.Title = "商机A"
	d1.Stage = model.StageLead
	s.Create(d1)

	d2 := model.NewDeal()
	d2.ID = model.DealID()
	d2.Slug = "deal-2"
	d2.Title = "商机B"
	d2.Stage = model.StageQualified
	s.Create(d2)

	results, err := s.List("lead", "", "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 lead deal, got %d", len(results))
	}
	if results[0].Title != "商机A" {
		t.Errorf("expected '商机A', got %q", results[0].Title)
	}
}

func TestDealList_StageNotIn(t *testing.T) {
	db := openTestDB(t)
	s := NewDealStore(db)

	d1 := model.NewDeal()
	d1.ID = model.DealID()
	d1.Slug = "deal-won"
	d1.Title = "已成交"
	d1.Stage = model.StageWon
	s.Create(d1)

	d2 := model.NewDeal()
	d2.ID = model.DealID()
	d2.Slug = "deal-lost"
	d2.Title = "已丢失"
	d2.Stage = model.StageLost
	s.Create(d2)

	d3 := model.NewDeal()
	d3.ID = model.DealID()
	d3.Slug = "deal-active"
	d3.Title = "活跃"
	d3.Stage = model.StageProposal
	s.Create(d3)

	results, err := s.List("", "won,lost", "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 active deal, got %d", len(results))
	}
}

func TestDealUpdateStage(t *testing.T) {
	db := openTestDB(t)
	s := NewDealStore(db)

	d := model.NewDeal()
	d.ID = model.DealID()
	d.Slug = "stage-test"
	d.Title = "阶段测试"
	d.Stage = model.StageLead
	s.Create(d)

	old, err := s.UpdateStage(d.ID, model.StageQualified, "测试", "bot1")
	if err != nil {
		t.Fatalf("UpdateStage: %v", err)
	}
	if old != model.StageLead {
		t.Errorf("expected old stage %q, got %q", model.StageLead, old)
	}

	got, _ := s.GetByID(d.ID)
	if got.Stage != model.StageQualified {
		t.Errorf("expected stage %q, got %q", model.StageQualified, got.Stage)
	}
}

func TestDealUpdateStage_WonIrreversible(t *testing.T) {
	db := openTestDB(t)
	s := NewDealStore(db)

	d := model.NewDeal()
	d.ID = model.DealID()
	d.Slug = "irrev-test"
	d.Title = "不可逆测试"
	d.Stage = model.StageWon
	s.Create(d)

	_, err := s.UpdateStage(d.ID, model.StageLead, "不应该允许", "bot1")
	if err == nil {
		t.Fatal("expected error when changing won stage")
	}
}

func TestDealDelete(t *testing.T) {
	db := openTestDB(t)
	s := NewDealStore(db)

	d := model.NewDeal()
	d.ID = model.DealID()
	d.Slug = "del-test"
	d.Title = "待删除"
	s.Create(d)

	err := s.Delete(d.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = s.GetByID(d.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

// --- ActivityStore ---

func TestActivityLog(t *testing.T) {
	db := openTestDB(t)
	s := NewActivityStore(db)

	a := model.NewActivity()
	a.ID = model.ActivityID()
	a.ContactIDs = []string{"cnt_001"}
	a.Type = "email"
	a.Summary = "测试活动"
	a.DedupeKey = "dedup:001"

	created, err := s.Log(a)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if !created {
		t.Error("expected created=true")
	}
}

func TestActivityLog_Dedup(t *testing.T) {
	db := openTestDB(t)
	s := NewActivityStore(db)

	a1 := model.NewActivity()
	a1.ID = model.ActivityID()
	a1.ContactIDs = []string{"cnt_002"}
	a1.Type = "email"
	a1.Summary = "不要重复"
	a1.DedupeKey = "unique-key"

	created, err := s.Log(a1)
	if err != nil {
		t.Fatalf("First Log: %v", err)
	}
	if !created {
		t.Error("expected created=true for first log")
	}

	a2 := model.NewActivity()
	a2.ID = model.ActivityID()
	a2.ContactIDs = []string{"cnt_002"}
	a2.Type = "email"
	a2.Summary = "不要重复"
	a2.DedupeKey = "unique-key"

	created, err = s.Log(a2)
	if err != nil {
		t.Fatalf("Second Log: %v", err)
	}
	if created {
		t.Error("expected created=false for duplicate")
	}
	if a2.ID != a1.ID {
		t.Errorf("expected same ID after dedup, got %q vs %q", a2.ID, a1.ID)
	}
}

func TestActivityListByContact(t *testing.T) {
	db := openTestDB(t)
	s := NewActivityStore(db)

	a := model.NewActivity()
	a.ID = model.ActivityID()
	a.ContactIDs = []string{"cnt_003"}
	a.Type = "call"
	a.Summary = "电话沟通"
	a.DedupeKey = "dedup:003"
	s.Log(a)

	results, err := s.ListByContact("cnt_003", "", "", 10)
	if err != nil {
		t.Fatalf("ListByContact: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(results))
	}
	if results[0].Summary != "电话沟通" {
		t.Errorf("expected '电话沟通', got %q", results[0].Summary)
	}
}

func TestActivitySearch_LIKE(t *testing.T) {
	db := openTestDB(t)
	s := NewActivityStore(db)

	a := model.NewActivity()
	a.ID = model.ActivityID()
	a.ContactIDs = []string{"cnt_004"}
	a.Type = "meeting"
	a.Summary = "项目启动会议"
	a.Body = "讨论了技术方案和排期"
	a.DedupeKey = "dedup:004"
	s.Log(a)

	results, err := s.Search("启动", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results for CJK query")
	}
}

// --- EventStore ---

func TestEventAppend(t *testing.T) {
	db := openTestDB(t)
	s := NewEventStore(db)

	seq, err := s.Append("bot1", "test.event", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	if seq <= 0 {
		t.Errorf("expected positive seq, got %d", seq)
	}
}

func TestEventPoll(t *testing.T) {
	db := openTestDB(t)
	s := NewEventStore(db)

	s.Append("bot1", "event.a", nil)
	s.Append("bot2", "event.b", nil)

	events, err := s.Poll(0, "", "", 10)
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestEventPoll_AfterSeq(t *testing.T) {
	db := openTestDB(t)
	s := NewEventStore(db)

	s.Append("bot1", "event.1", nil)
	seq, _ := s.Append("bot1", "event.2", nil)

	events, err := s.Poll(seq, "", "", 10)
	if err != nil {
		t.Fatalf("Poll after seq: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events after last seq, got %d", len(events))
	}
}

func TestEventPoll_ExcludeActor(t *testing.T) {
	db := openTestDB(t)
	s := NewEventStore(db)

	s.Append("bot1", "event.1", nil)
	s.Append("bot2", "event.2", nil)

	events, err := s.Poll(0, "", "bot1", 10)
	if err != nil {
		t.Fatalf("Poll exclude: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event after excluding bot1, got %d", len(events))
	}
	if events[0].Actor != "bot2" {
		t.Errorf("expected bot2 event, got actor %q", events[0].Actor)
	}
}

func TestEventGetLastSeq(t *testing.T) {
	db := openTestDB(t)
	s := NewEventStore(db)

	seq, _ := s.GetLastSeq()
	if seq != 0 {
		t.Errorf("expected 0 for empty table, got %d", seq)
	}

	s.Append("bot1", "event.1", nil)

	seq, _ = s.GetLastSeq()
	if seq != 1 {
		t.Errorf("expected 1, got %d", seq)
	}
}

// --- MemoStore ---

func TestMemoInsert(t *testing.T) {
	db := openTestDB(t)
	s := NewMemoStore(db, tempDir(t))

	m := &model.Memo{
		ID:        model.MemoID(),
		ScopeType: "contact",
		ScopeID:   "cnt_001",
		Text:      "测试记忆",
		Decay:     "180d",
	}

	err := s.Insert(m)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
}

func TestMemoListByScope(t *testing.T) {
	db := openTestDB(t)
	s := NewMemoStore(db, tempDir(t))

	m1 := &model.Memo{ID: model.MemoID(), ScopeType: "contact", ScopeID: "cnt_001", Text: "记忆A"}
	m2 := &model.Memo{ID: model.MemoID(), ScopeType: "contact", ScopeID: "cnt_001", Text: "记忆B"}
	m3 := &model.Memo{ID: model.MemoID(), ScopeType: "deal", ScopeID: "del_001", Text: "商机记忆"}
	s.Insert(m1)
	s.Insert(m2)
	s.Insert(m3)

	memos, err := s.ListByScope("contact", "cnt_001")
	if err != nil {
		t.Fatalf("ListByScope: %v", err)
	}
	if len(memos) != 2 {
		t.Fatalf("expected 2 contact memos, got %d", len(memos))
	}

	memos, err = s.ListByScope("deal", "del_001")
	if err != nil {
		t.Fatalf("ListByScope deal: %v", err)
	}
	if len(memos) != 1 {
		t.Fatalf("expected 1 deal memo, got %d", len(memos))
	}
}

func TestMemoMarkExpired(t *testing.T) {
	db := openTestDB(t)
	s := NewMemoStore(db, tempDir(t))

	m := &model.Memo{ID: model.MemoID(), ScopeType: "contact", ScopeID: "cnt_001", Text: "将过期"}
	s.Insert(m)

	err := s.MarkExpired(m.ID)
	if err != nil {
		t.Fatalf("MarkExpired: %v", err)
	}

	got, err := s.GetByID(m.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if !got.Expired {
		t.Error("expected expired=true after MarkExpired")
	}
}

func TestMemoGetByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	s := NewMemoStore(db, tempDir(t))

	_, err := s.GetByID("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent memo")
	}
}

func TestMemoDelete(t *testing.T) {
	db := openTestDB(t)
	s := NewMemoStore(db, tempDir(t))

	m := &model.Memo{ID: model.MemoID(), ScopeType: "contact", ScopeID: "cnt_001", Text: "待删除"}
	s.Insert(m)

	err := s.Delete(m.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = s.GetByID(m.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestTextSimilarity_Identical(t *testing.T) {
	if TextSimilarity("hello world", "hello world") != 1.0 {
		t.Error("expected 1.0 for identical strings")
	}
}

func TestTextSimilarity_Empty(t *testing.T) {
	if TextSimilarity("", "hello") != 0.0 {
		t.Error("expected 0.0 when one string is empty")
	}
}

func TestTextSimilarity_Partial(t *testing.T) {
	sim := TextSimilarity("喜欢喝咖啡", "喜欢喝茶")
	if sim <= 0 {
		t.Error("expected positive similarity for partial match")
	}
	if sim >= 1.0 {
		t.Error("expected less than 1.0 for partial match")
	}
}

func TestMemoDecayScan(t *testing.T) {
	db := openTestDB(t)
	s := NewMemoStore(db, tempDir(t))

	// 一个刚创建的 memo（不过期）
	m1 := &model.Memo{ID: model.MemoID(), ScopeType: "contact", ScopeID: "cnt_001", Text: "新鲜的", Decay: "180d"}
	s.Insert(m1)

	// 一个已过期的 memo
	m2 := &model.Memo{ID: model.MemoID(), ScopeType: "contact", ScopeID: "cnt_001",
		Text: "过期的", ValidFrom: "2020-01-01T00:00:00Z", Decay: "30d"}
	s.Insert(m2)

	count, err := s.DecayScan(false)
	if err != nil {
		t.Fatalf("DecayScan: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 expired, got %d", count)
	}

	got, _ := s.GetByID(m2.ID)
	if !got.Expired {
		t.Error("expected m2 to be marked expired")
	}
}

func TestMemoDecayScan_DryRun(t *testing.T) {
	db := openTestDB(t)
	s := NewMemoStore(db, tempDir(t))

	m := &model.Memo{ID: model.MemoID(), ScopeType: "contact", ScopeID: "cnt_001",
		Text: "预览过期", ValidFrom: "2020-01-01T00:00:00Z", Decay: "30d"}
	s.Insert(m)

	count, err := s.DecayScan(true)
	if err != nil {
		t.Fatalf("DecayScan dry-run: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 found in dry-run, got %d", count)
	}

	got, _ := s.GetByID(m.ID)
	if got.Expired {
		t.Error("expected not expired after dry-run")
	}
}

func TestMemoPropose_Clean(t *testing.T) {
	dir := tempDir(t)
	db := openTestDB(t)
	s := NewMemoStore(db, dir)

	p, err := s.Propose("contact:cnt_001", "全新的事实陈述", "来源", "bot1", 0.9)
	if err != nil {
		t.Fatalf("Propose: %v", err)
	}
	if p.Status != "clean" {
		t.Errorf("expected status 'clean', got %q", p.Status)
	}
	if len(p.ConflictWith) != 0 {
		t.Errorf("expected 0 conflicts, got %d", len(p.ConflictWith))
	}
}

func TestTextSimilarity_Threshold(t *testing.T) {
	// bigram Jaccard: 4 shared bigrams vs large union — naturally < 0.5
	a := "喜欢喝咖啡"
	b := "喜欢喝咖啡，每天下午都要一杯美式"
	sim := TextSimilarity(a, b)
	if sim <= 0.0 {
		t.Errorf("expected some similarity, got %f", sim)
	}
	if sim >= 1.0 {
		t.Errorf("expected partial similarity, got %f", sim)
	}
}

// --- FileStore basics (same package, test directly) ---

func TestFileStore_Init(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)

	err := fs.Init()
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	dirs := []string{"contacts", "deals", "activities", "events", ".subscribers", "proposals", "alerts", "rules"}
	for _, d := range dirs {
		path := filepath.Join(dir, d)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected directory %s to exist", path)
		}
	}
}

func TestFileStore_WriteReadContact(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)
	os.MkdirAll(filepath.Join(dir, "contacts"), 0755)

	c := model.NewContact()
	c.ID = model.ContactID()
	c.Slug = "test-slug"
	c.Name = "文件测试"
	c.Emails = []string{"file@test.com"}
	c.Company = "文件公司"
	c.Body = "这是正文"

	err := fs.WriteContact(c)
	if err != nil {
		t.Fatalf("WriteContact: %v", err)
	}

	got, err := fs.ReadContact("test-slug")
	if err != nil {
		t.Fatalf("ReadContact: %v", err)
	}
	if got.Name != "文件测试" {
		t.Errorf("expected name '文件测试', got %q", got.Name)
	}
	if got.Body != "这是正文" {
		t.Errorf("expected body '这是正文', got %q", got.Body)
	}
	if got.Company != "文件公司" {
		t.Errorf("expected company '文件公司', got %q", got.Company)
	}
}

func TestFileStore_ContactListFiles(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)
	if err := fs.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	c1 := model.NewContact()
	c1.ID = model.ContactID()
	c1.Slug = "slug-a"
	c1.Name = "A"
	fs.WriteContact(c1)

	c2 := model.NewContact()
	c2.ID = model.ContactID()
	c2.Slug = "slug-b"
	c2.Name = "B"
	fs.WriteContact(c2)

	slugs, err := fs.ListContactFiles()
	if err != nil {
		t.Fatalf("ListContactFiles: %v", err)
	}
	if len(slugs) != 2 {
		t.Fatalf("expected 2 slugs, got %d", len(slugs))
	}
}

func TestFileStore_SubscriberCursor(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)
	os.MkdirAll(filepath.Join(dir, ".subscribers"), 0755)

	sc := &model.SubscriberCursor{
		Actor:   "test-bot",
		LastSeq: 42,
		Filter:  "type=contact.created",
	}

	err := fs.WriteSubscriberCursor(sc)
	if err != nil {
		t.Fatalf("WriteSubscriberCursor: %v", err)
	}

	got, err := fs.ReadSubscriberCursor("test-bot")
	if err != nil {
		t.Fatalf("ReadSubscriberCursor: %v", err)
	}
	if got.Actor != "test-bot" {
		t.Errorf("expected actor 'test-bot', got %q", got.Actor)
	}
	if got.LastSeq != 42 {
		t.Errorf("expected LastSeq 42, got %d", got.LastSeq)
	}
}

func TestFileStore_SubscriberCursor_Defaults(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)

	sc, err := fs.ReadSubscriberCursor("new-bot")
	if err != nil {
		t.Fatalf("ReadSubscriberCursor new: %v", err)
	}
	if sc.LastSeq != 0 {
		t.Errorf("expected LastSeq 0 for new subscriber, got %d", sc.LastSeq)
	}
}

func TestFileStore_WriteReadDeal(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)
	if err := fs.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	d := model.NewDeal()
	d.ID = model.DealID()
	d.Slug = "test-deal"
	d.Title = "Test Deal"
	d.Stage = "lead"
	d.Amount = 100000
	if err := fs.WriteDeal(d); err != nil {
		t.Fatalf("WriteDeal: %v", err)
	}

	got, err := fs.ReadDeal("test-deal")
	if err != nil {
		t.Fatalf("ReadDeal: %v", err)
	}
	if got.ID != d.ID {
		t.Errorf("expected ID %q, got %q", d.ID, got.ID)
	}
	if got.Title != "Test Deal" {
		t.Errorf("expected Title 'Test Deal', got %q", got.Title)
	}
	if got.Amount != 100000 {
		t.Errorf("expected Amount 100000, got %d", got.Amount)
	}
}

func TestFileStore_DeleteDeal(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)
	if err := fs.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	d := model.NewDeal()
	d.ID = model.DealID()
	d.Slug = "delete-deal"
	d.Title = "to delete"
	if err := fs.WriteDeal(d); err != nil {
		t.Fatalf("WriteDeal: %v", err)
	}
	if err := fs.DeleteDeal("delete-deal"); err != nil {
		t.Fatalf("DeleteDeal: %v", err)
	}
	if _, err := fs.ReadDeal("delete-deal"); err == nil {
		t.Error("expected error reading deleted deal")
	}
}

func TestFileStore_WriteReadConfig(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)
	if err := fs.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	cfg := model.DefaultConfig(dir)
	if err := fs.WriteConfig(cfg); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	got, err := fs.ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	if got.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", got.Version)
	}
	if got.User.Timezone != "Asia/Shanghai" {
		t.Errorf("expected timezone 'Asia/Shanghai', got %q", got.User.Timezone)
	}
}

func TestFileStore_AppendReadActivities(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)
	if err := fs.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	a1 := model.NewActivity()
	a1.ID = model.ActivityID()
	a1.Type = "call"
	a1.Summary = "First call"
	if err := fs.AppendActivity(a1); err != nil {
		t.Fatalf("AppendActivity: %v", err)
	}

	a2 := model.NewActivity()
	a2.ID = model.ActivityID()
	a2.Type = "email"
	a2.Summary = "Follow-up email"
	if err := fs.AppendActivity(a2); err != nil {
		t.Fatalf("AppendActivity: %v", err)
	}

	activities, err := fs.ReadActivities(Now()[:7])
	if err != nil {
		t.Fatalf("ReadActivities: %v", err)
	}
	if len(activities) != 2 {
		t.Fatalf("expected 2 activities, got %d", len(activities))
	}
	if activities[0].Type != "call" {
		t.Errorf("expected first activity type 'call', got %q", activities[0].Type)
	}
	if activities[1].Type != "email" {
		t.Errorf("expected second activity type 'email', got %q", activities[1].Type)
	}
}

func TestFileStore_WriteReadContactMemory(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)
	if err := fs.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	if err := fs.WriteContactMemory("test-contact", "这是测试记忆"); err != nil {
		t.Fatalf("WriteContactMemory: %v", err)
	}

	text, err := fs.ReadContactMemory("test-contact")
	if err != nil {
		t.Fatalf("ReadContactMemory: %v", err)
	}
	if text != "这是测试记忆" {
		t.Errorf("expected '这是测试记忆', got %q", text)
	}

	// 追加
	if err := fs.WriteContactMemory("test-contact", "第二行记忆"); err != nil {
		t.Fatalf("WriteContactMemory append: %v", err)
	}
	text, err = fs.ReadContactMemory("test-contact")
	if err != nil {
		t.Fatalf("ReadContactMemory after append: %v", err)
	}
	if !strings.Contains(text, "第二行记忆") {
		t.Errorf("expected appended text, got %q", text)
	}
}

func TestDealUpdateField(t *testing.T) {
	db := openTestDB(t)
	store := NewDealStore(db)

	d := model.NewDeal()
	d.ID = model.DealID()
	d.Slug = "update-field"
	d.Title = "Original"
	d.Amount = 50000
	if err := store.Create(d); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.UpdateField(d.ID, "amount", 100000); err != nil {
		t.Fatalf("UpdateField: %v", err)
	}

	got, err := store.GetByID(d.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Amount != 100000 {
		t.Errorf("expected Amount 100000, got %d", got.Amount)
	}
}

func TestDealUpdateField_SameValue(t *testing.T) {
	db := openTestDB(t)
	store := NewDealStore(db)

	d := model.NewDeal()
	d.ID = model.DealID()
	d.Slug = "update-field-same"
	d.Title = "Same"
	d.Amount = 50000
	if err := store.Create(d); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.UpdateField(d.ID, "amount", 50000); err != nil {
		t.Fatalf("UpdateField: %v", err)
	}
}

func TestEventAppend_WithPayload(t *testing.T) {
	db := openTestDB(t)
	store := NewEventStore(db)

	seq, err := store.Append("test-bot", "contact.created", map[string]string{"contact_id": "c1"})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	if seq <= 0 {
		t.Errorf("expected Seq > 0, got %d", seq)
	}
}

func TestActivityListSince(t *testing.T) {
	db := openTestDB(t)
	store := NewActivityStore(db)

	d1 := model.NewActivity()
	d1.ID = model.ActivityID()
	d1.DedupeKey = "list-since-1"
	d1.Type = "note"
	d1.ContactIDs = []string{"c1"}
	d1.Summary = "first"
	if _, err := store.Log(d1); err != nil {
		t.Fatalf("Log: %v", err)
	}

	time.Sleep(2 * time.Millisecond)

	d2 := model.NewActivity()
	d2.ID = model.ActivityID()
	d2.DedupeKey = "list-since-2"
	d2.Type = "note"
	d2.ContactIDs = []string{"c1"}
	d2.Summary = "second"
	if _, err := store.Log(d2); err != nil {
		t.Fatalf("Log: %v", err)
	}

	results, err := store.ListSince(d1.Timestamp, 10)
	if err != nil {
		t.Fatalf("ListSince: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}
	if results[0].ID != d2.ID {
		t.Errorf("expected %q (second), got %q", d2.ID, results[0].ID)
	}
}

func TestMemoCommit_NotFound(t *testing.T) {
	db := openTestDB(t)
	dir := tempDir(t)
	store := NewMemoStore(db, dir)

	// committing a non-existent proposal should error
	err := store.Commit("nonexistent-proposal", "supersede")
	if err == nil {
		t.Error("expected error for non-existent proposal")
	}
}

func TestFileStore_DeleteContact(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)
	if err := fs.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	c := model.NewContact()
	c.ID = model.ContactID()
	c.Slug = "delete-me"
	c.Name = "Delete Me"
	if err := fs.WriteContact(c); err != nil {
		t.Fatalf("WriteContact: %v", err)
	}

	if err := fs.DeleteContact("delete-me", ""); err != nil {
		t.Fatalf("DeleteContact: %v", err)
	}
	if _, err := fs.ReadContact("delete-me"); err == nil {
		t.Error("expected error reading deleted contact")
	}
}

func TestFileStore_AppendEvent(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)
	if err := fs.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	e := &model.Event{
		TS:    Now(),
		Actor: "test-bot",
		Type:  "test.event",
	}
	if err := fs.AppendEvent(e); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}
}

func TestFileStore_AppendProposal(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)
	if err := fs.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	p := &model.Proposal{
		ID:        model.NewID("prop"),
		Scope:     "contact:c1",
		Statement: "测试提议",
		Status:    "pending",
	}
	if err := fs.AppendProposal(p); err != nil {
		t.Fatalf("AppendProposal: %v", err)
	}
}

func TestFileStore_WriteReadPendingAlerts(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)
	if err := fs.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	alerts := []model.Alert{{
		ID:       model.NewID(model.PrefixAlert),
		RuleName: "test_rule",
		Title:    "Test Alert",
		Status:   model.AlertStatusPending,
		Priority: 1,
	}}
	if err := fs.WritePendingAlerts(alerts); err != nil {
		t.Fatalf("WritePendingAlerts: %v", err)
	}

	got, err := fs.ReadPendingAlerts()
	if err != nil {
		t.Fatalf("ReadPendingAlerts: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(got))
	}
	if got[0].RuleName != "test_rule" {
		t.Errorf("expected rule 'test_rule', got %q", got[0].RuleName)
	}
}

func TestStoreOpenClose(t *testing.T) {
	dir := tempDir(t)

	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if s.ConfigDir() != dir {
		t.Errorf("expected ConfigDir %q, got %q", dir, s.ConfigDir())
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// --- FileStore: newly added list methods ---

func TestFileStore_ListDealFiles(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)
	if err := fs.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	d1 := model.NewDeal()
	d1.ID = model.DealID()
	d1.Slug = "deal-alpha"
	d1.Title = "Alpha"
	if err := fs.WriteDeal(d1); err != nil {
		t.Fatalf("WriteDeal: %v", err)
	}

	d2 := model.NewDeal()
	d2.ID = model.DealID()
	d2.Slug = "deal-beta"
	d2.Title = "Beta"
	if err := fs.WriteDeal(d2); err != nil {
		t.Fatalf("WriteDeal: %v", err)
	}

	files, err := fs.ListDealFiles()
	if err != nil {
		t.Fatalf("ListDealFiles: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}

func TestFileStore_ListActivityFiles(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)
	if err := fs.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	a := model.NewActivity()
	a.ID = model.ActivityID()
	a.Type = "note"
	a.Summary = "test"
	if err := fs.AppendActivity(a); err != nil {
		t.Fatalf("AppendActivity: %v", err)
	}

	files, err := fs.ListActivityFiles()
	if err != nil {
		t.Fatalf("ListActivityFiles: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
}

func TestFileStore_ListDealFiles_EmptyDir(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)
	// no init — directory doesn't exist

	files, err := fs.ListDealFiles()
	if err != nil {
		t.Fatalf("ListDealFiles empty: %v", err)
	}
	if files != nil {
		t.Errorf("expected nil for empty list, got %v", files)
	}
}

func TestFileStore_ListActivityFiles_EmptyDir(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)

	files, err := fs.ListActivityFiles()
	if err != nil {
		t.Fatalf("ListActivityFiles empty: %v", err)
	}
	if files != nil {
		t.Errorf("expected nil for empty list, got %v", files)
	}
}

// --- FileStore: read edge cases ---

func TestFileStore_ReadContactMemory_NotFound(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)
	if err := fs.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	text, err := fs.ReadContactMemory("nonexistent")
	if err != nil {
		t.Fatalf("ReadContactMemory not found: %v", err)
	}
	if text != "" {
		t.Errorf("expected empty string, got %q", text)
	}
}

func TestFileStore_ReadPendingAlerts_NotFound(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)
	if err := fs.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	alerts, err := fs.ReadPendingAlerts()
	if err != nil {
		t.Fatalf("ReadPendingAlerts not found: %v", err)
	}
	if alerts != nil {
		t.Errorf("expected nil, got %v", alerts)
	}
}

func TestFileStore_ReadConfig_NotFound(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)
	if err := fs.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	cfg, err := fs.ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig not found: %v", err)
	}
	if cfg.Version != "1.0.0" {
		t.Errorf("expected default version, got %q", cfg.Version)
	}
}

func TestFileStore_ReadActivities_NotFound(t *testing.T) {
	dir := tempDir(t)
	fs := NewFileStore(dir)
	if err := fs.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	activities, err := fs.ReadActivities("2099-01")
	if err != nil {
		t.Fatalf("ReadActivities not found: %v", err)
	}
	if activities != nil {
		t.Errorf("expected nil, got %v", activities)
	}
}

func TestContactStore_Search_NoResults(t *testing.T) {
	db := openTestDB(t)
	s := NewContactStore(db)

	results, err := s.Search("这个查询不应该匹配任何联系人", 10)
	if err != nil {
		t.Fatalf("Search no results: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestDealStore_List_Owner(t *testing.T) {
	db := openTestDB(t)
	s := NewDealStore(db)

	d := model.NewDeal()
	d.ID = model.DealID()
	d.Slug = "owner-deal"
	d.Title = "Owner Deal"
	d.Owner = "agent-1"
	if err := s.Create(d); err != nil {
		t.Fatalf("Create: %v", err)
	}

	results, err := s.List("", "", "agent-1")
	if err != nil {
		t.Fatalf("List by owner: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 deal for owner, got %d", len(results))
	}

	results, err = s.List("", "", "other-agent")
	if err != nil {
		t.Fatalf("List by other: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 deals for other owner, got %d", len(results))
	}
}

func TestEventStore_Poll_Filter(t *testing.T) {
	db := openTestDB(t)
	s := NewEventStore(db)

	s.Append("bot1", "contact.created", nil)
	s.Append("bot1", "deal.created", nil)
	s.Append("bot1", "activity.logged", nil)

	events, err := s.Poll(0, "type=contact.created", "", 10)
	if err != nil {
		t.Fatalf("Poll filtered: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 filtered event, got %d", len(events))
	}
	if events[0].Type != "contact.created" {
		t.Errorf("expected contact.created, got %q", events[0].Type)
	}
}
