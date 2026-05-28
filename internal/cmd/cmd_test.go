package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/AgentPal/AgentCRM/internal/model"
)

// executeCommand 执行命令并捕获 stdout 输出。
func executeCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()

	// 重置包级变量，防止跨调用泄漏
	format = "text"
	actor = ""
	// 重置子命令标志，防止 cobra 跨调用泄漏
	doctorCmd.Flags().Set("check", "all")
	doctorCmd.Flags().Set("fix", "false")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs(args)
	err := rootCmd.Execute()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return strings.TrimSpace(buf.String()), err
}

func TestInit(t *testing.T) {
	dir := t.TempDir()
	out, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "已初始化") {
		t.Errorf("unexpected output: %s", out)
	}

	// Verify directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("data dir not created")
	}
	if _, err := os.Stat(dir + "/index.db"); os.IsNotExist(err) {
		t.Errorf("index.db not created")
	}
}

func TestVersion(t *testing.T) {
	out, err := executeCommand(t, "version")
	if err != nil {
		t.Fatalf("version failed: %v", err)
	}
	if !strings.Contains(out, "AgentCRM") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestContactLifecycle(t *testing.T) {
	dir := t.TempDir()

	// init
	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// upsert
	out, err := executeCommand(t, "--data-dir", dir, "--format", "json",
		"contact", "upsert",
		"--name", "张三",
		"--email", "zhangsan@example.com",
		"--company", "ACME Corp",
		"--title", "CTO",
		"--phone", "13800138000",
		"--tag", "vip",
		"--tag", "tech",
		"--source", "manual")
	if err != nil {
		t.Fatalf("upsert failed: %v\noutput: %s", err, out)
	}
	var created struct {
		ID      string `json:"id"`
		Slug    string `json:"slug"`
		Created bool   `json:"created"`
	}
	if err := json.Unmarshal([]byte(out), &created); err != nil {
		t.Fatalf("parse output: %v\nraw: %s", err, out)
	}
	if !created.Created {
		t.Error("expected created=true")
	}
	if created.ID == "" {
		t.Error("expected non-empty ID")
	}
	contactID := created.ID

	// get
	out, err = executeCommand(t, "--data-dir", dir, "--format", "json",
		"contact", "get", contactID)
	if err != nil {
		t.Fatalf("get failed: %v\noutput: %s", err, out)
	}
	var c model.Contact
	if err := json.Unmarshal([]byte(out), &c); err != nil {
		t.Fatalf("parse contact: %v\nraw: %s", err, out)
	}
	if c.Name != "张三" {
		t.Errorf("expected 张三, got %s", c.Name)
	}
	if c.Company != "ACME Corp" {
		t.Errorf("expected ACME Corp, got %s", c.Company)
	}
	if len(c.Emails) == 0 || c.Emails[0] != "zhangsan@example.com" {
		t.Errorf("unexpected emails: %v", c.Emails)
	}

	// list
	out, err = executeCommand(t, "--data-dir", dir, "--format", "json",
		"contact", "list")
	if err != nil {
		t.Fatalf("list failed: %v\noutput: %s", err, out)
	}
	var results []model.ContactSearchResult
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("parse list: %v\nraw: %s", err, out)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 contact, got %d", len(results))
	}

	// update
	out, err = executeCommand(t, "--data-dir", dir, "--format", "json",
		"contact", "update", contactID,
		"--set", "company=New Corp",
		"--reason", "公司更名")
	if err != nil {
		t.Fatalf("update failed: %v\noutput: %s", err, out)
	}

	// verify update via get
	out, err = executeCommand(t, "--data-dir", dir, "--format", "json",
		"contact", "get", contactID)
	if err != nil {
		t.Fatalf("get after update failed: %v", err)
	}
	if err := json.Unmarshal([]byte(out), &c); err != nil {
		t.Fatalf("parse contact: %v", err)
	}
	if c.Company != "New Corp" {
		t.Errorf("expected New Corp, got %s", c.Company)
	}

	// history
	out, err = executeCommand(t, "--data-dir", dir, "--format", "json",
		"contact", "history", contactID,
		"--field", "company")
	if err != nil {
		t.Fatalf("history failed: %v\noutput: %s", err, out)
	}
	var history []model.FieldHistory
	if err := json.Unmarshal([]byte(out), &history); err != nil {
		t.Fatalf("parse history: %v\nraw: %s", err, out)
	}
	if len(history) == 0 {
		t.Error("expected at least 1 history entry")
	}

	// create second contact for merge
	out, err = executeCommand(t, "--data-dir", dir, "--format", "json",
		"contact", "upsert",
		"--name", "李四",
		"--email", "lisi@example.com",
		"--company", "Old Corp")
	if err != nil {
		t.Fatalf("upsert lisi failed: %v", err)
	}
	var lisi struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(out), &lisi)

	// merge
	out, err = executeCommand(t, "--data-dir", dir,
		"contact", "merge", contactID, lisi.ID,
		"--reason", "重复联系人")
	if err != nil {
		t.Fatalf("merge failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "已合并") {
		t.Errorf("unexpected merge output: %s", out)
	}
}

func TestDealLifecycle(t *testing.T) {
	dir := t.TempDir()

	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// create a contact first
	out, err := executeCommand(t, "--data-dir", dir, "--format", "json",
		"contact", "upsert",
		"--name", "王五",
		"--email", "wangwu@example.com")
	if err != nil {
		t.Fatalf("create contact failed: %v", err)
	}
	var contact struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(out), &contact)

	// create deal
	out, err = executeCommand(t, "--data-dir", dir, "--format", "json",
		"deal", "create",
		"--title", "企业版订阅",
		"--contact", contact.ID,
		"--amount", "50000",
		"--stage", "qualified")
	if err != nil {
		t.Fatalf("create deal failed: %v\noutput: %s", err, out)
	}
	var dealData struct {
		ID    string `json:"id"`
		Slug  string `json:"slug"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(out), &dealData); err != nil {
		t.Fatalf("parse deal: %v\nraw: %s", err, out)
	}
	dealID := dealData.ID

	// get deal
	out, err = executeCommand(t, "--data-dir", dir, "--format", "json",
		"deal", "get", dealID)
	if err != nil {
		t.Fatalf("get deal failed: %v\noutput: %s", err, out)
	}
	var d model.Deal
	if err := json.Unmarshal([]byte(out), &d); err != nil {
		t.Fatalf("parse deal: %v\nraw: %s", err, out)
	}
	if d.Title != "企业版订阅" {
		t.Errorf("expected 企业版订阅, got %s", d.Title)
	}
	if d.Amount != 50000 {
		t.Errorf("expected 50000, got %d", d.Amount)
	}
	if d.Stage != "qualified" {
		t.Errorf("expected qualified, got %s", d.Stage)
	}

	// update stage
	out, err = executeCommand(t, "--data-dir", dir,
		"deal", "update", dealID,
		"--set", "stage=proposal",
		"--reason", "已发送方案")
	if err != nil {
		t.Fatalf("update deal stage failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "qualified → proposal") {
		t.Errorf("unexpected update output: %s", out)
	}

	// update amount
	out, err = executeCommand(t, "--data-dir", dir,
		"deal", "update", dealID,
		"--set", "amount=60000",
		"--reason", "增加了用户数")
	if err != nil {
		t.Fatalf("update deal amount failed: %v\noutput: %s", err, out)
	}

	// list deals
	out, err = executeCommand(t, "--data-dir", dir, "--format", "json",
		"deal", "list")
	if err != nil {
		t.Fatalf("list deals failed: %v\noutput: %s", err, out)
	}
	var deals []model.DealSearchResult
	if err := json.Unmarshal([]byte(out), &deals); err != nil {
		t.Fatalf("parse deals: %v\nraw: %s", err, out)
	}
	if len(deals) != 1 {
		t.Errorf("expected 1 deal, got %d", len(deals))
	}

	// history
	out, err = executeCommand(t, "--data-dir", dir, "--format", "json",
		"deal", "history", dealID,
		"--field", "stage")
	if err != nil {
		t.Fatalf("deal history failed: %v\noutput: %s", err, out)
	}
	var stageHistory []model.StageHistory
	if err := json.Unmarshal([]byte(out), &stageHistory); err != nil {
		t.Fatalf("parse stage history: %v\nraw: %s", err, out)
	}
	if len(stageHistory) == 0 {
		t.Error("expected stage history entries")
	}

	// summarize
	out, err = executeCommand(t, "--data-dir", dir,
		"deal", "summarize", dealID)
	if err != nil {
		t.Fatalf("summarize failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "企业版订阅") {
		t.Errorf("expected deal title in summary, got: %s", out)
	}
}

func TestActivityLifecycle(t *testing.T) {
	dir := t.TempDir()

	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// create contact
	out, err := executeCommand(t, "--data-dir", dir, "--format", "json",
		"contact", "upsert",
		"--name", "测试联系人",
		"--email", "test@example.com")
	if err != nil {
		t.Fatalf("create contact failed: %v", err)
	}
	var contact struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(out), &contact)

	// log activity
	out, err = executeCommand(t, "--data-dir", dir, "--format", "json",
		"activity", "log",
		"--contact", contact.ID,
		"--type", "email",
		"--direction", "out",
		"--channel", "gmail",
		"--summary", "发送了产品介绍",
		"--dedupe-key", "email-001")
	if err != nil {
		t.Fatalf("log activity failed: %v\noutput: %s", err, out)
	}
	var actData struct {
		ID      string `json:"id"`
		Created bool   `json:"created"`
	}
	if err := json.Unmarshal([]byte(out), &actData); err != nil {
		t.Fatalf("parse activity: %v\nraw: %s", err, out)
	}
	if !actData.Created {
		t.Error("expected created=true")
	}

	// log a second activity
	_, err = executeCommand(t, "--data-dir", dir,
		"activity", "log",
		"--contact", contact.ID,
		"--type", "call",
		"--direction", "in",
		"--summary", "电话沟通需求",
		"--dedupe-key", "call-001")
	if err != nil {
		t.Fatalf("log second activity failed: %v", err)
	}

	// list activities
	out, err = executeCommand(t, "--data-dir", dir, "--format", "json",
		"activity", "list",
		"--contact", contact.ID)
	if err != nil {
		t.Fatalf("list activities failed: %v\noutput: %s", err, out)
	}
	var activities []model.Activity
	if err := json.Unmarshal([]byte(out), &activities); err != nil {
		t.Fatalf("parse activities: %v\nraw: %s", err, out)
	}
	if len(activities) < 2 {
		t.Errorf("expected at least 2 activities, got %d", len(activities))
	}

	// timeline
	out, err = executeCommand(t, "--data-dir", dir,
		"activity", "timeline", contact.ID)
	if err != nil {
		t.Fatalf("timeline failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "产品介绍") && !strings.Contains(out, "电话") {
		t.Errorf("expected activity in timeline, got: %s", out)
	}

	// dedupe - log same activity again should return existing
	out, err = executeCommand(t, "--data-dir", dir, "--format", "json",
		"activity", "log",
		"--contact", contact.ID,
		"--type", "email",
		"--summary", "发送了产品介绍",
		"--dedupe-key", "email-001")
	if err != nil {
		t.Fatalf("log duplicate failed: %v", err)
	}
	var dupData struct {
		Created bool `json:"created"`
	}
	json.Unmarshal([]byte(out), &dupData)
	if dupData.Created {
		t.Error("expected created=false for duplicate")
	}
}

func TestMemoryLifecycle(t *testing.T) {
	dir := t.TempDir()

	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// create a contact first
	out, err := executeCommand(t, "--data-dir", dir, "--format", "json",
		"contact", "upsert",
		"--name", "记忆测试",
		"--email", "memory@example.com")
	if err != nil {
		t.Fatalf("create contact failed: %v", err)
	}
	var contact struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(out), &contact)
	scope := "contact:" + contact.ID

	// write memory
	out, err = executeCommand(t, "--data-dir", dir, "--format", "json",
		"memory", "write",
		"--scope", scope,
		"--text", "该客户偏好邮件沟通",
		"--decay", "30d")
	if err != nil {
		t.Fatalf("write memory failed: %v\noutput: %s", err, out)
	}
	var memoResult struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(out), &memoResult); err != nil {
		t.Fatalf("parse memo: %v\nraw: %s", err, out)
	}
	if memoResult.ID == "" {
		t.Error("expected non-empty memo ID")
	}

	// list memory
	out, err = executeCommand(t, "--data-dir", dir, "--format", "json",
		"memory", "list",
		"--scope", scope)
	if err != nil {
		t.Fatalf("list memory failed: %v\noutput: %s", err, out)
	}
	var memos []model.Memo
	if err := json.Unmarshal([]byte(out), &memos); err != nil {
		t.Fatalf("parse memos: %v\nraw: %s", err, out)
	}
	if len(memos) != 1 {
		t.Errorf("expected 1 memo, got %d", len(memos))
	}

	// recall memory
	out, err = executeCommand(t, "--data-dir", dir, "--format", "json",
		"memory", "recall", "邮件",
		"--scope", scope,
		"--top-k", "5")
	if err != nil {
		t.Fatalf("recall failed: %v\noutput: %s", err, out)
	}
	var results []struct {
		ID    string  `json:"id"`
		Text  string  `json:"text"`
		Score float64 `json:"score"`
	}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Logf("recall output (may be empty): %s", out)
	}

	// forget memory
	out, err = executeCommand(t, "--data-dir", dir,
		"memory", "forget", memoResult.ID)
	if err != nil {
		t.Fatalf("forget failed: %v\noutput: %s", err, out)
	}
}

func TestEvents(t *testing.T) {
	dir := t.TempDir()

	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// create a contact (generates contact.created event)
	_, err = executeCommand(t, "--data-dir", dir, "--format", "json",
		"contact", "upsert",
		"--name", "事件测试",
		"--email", "event@example.com")
	if err != nil {
		t.Fatalf("create contact failed: %v", err)
	}

	// poll events as different actor
	out, err := executeCommand(t, "--data-dir", dir, "--format", "json",
		"events", "poll",
		"--as", "test-actor",
		"--include-self")
	if err != nil {
		t.Fatalf("poll events failed: %v\noutput: %s", err, out)
	}
	var events []model.Event
	if err := json.Unmarshal([]byte(out), &events); err != nil {
		t.Fatalf("parse events: %v\nraw: %s", err, out)
	}
	if len(events) == 0 {
		t.Error("expected at least 1 event")
	}

	// ack events
	lastSeq := int64(0)
	if len(events) > 0 {
		lastSeq = events[len(events)-1].Seq
	}
	out, err = executeCommand(t, "--data-dir", dir,
		"events", "ack",
		"--as", "test-actor",
		"--up-to-seq", "999")
	if err != nil {
		t.Fatalf("ack failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "已推进") {
		t.Errorf("unexpected ack output: %s", out)
	}
	_ = lastSeq
}

func TestExportJSON(t *testing.T) {
	dir := t.TempDir()

	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// create some data
	_, err = executeCommand(t, "--data-dir", dir, "--format", "json",
		"contact", "upsert",
		"--name", "导出测试",
		"--email", "export@example.com",
		"--company", "TestCorp")
	if err != nil {
		t.Fatalf("create contact failed: %v", err)
	}

	exportDir := dir + "/export"
	out, err := executeCommand(t, "--data-dir", dir,
		"export",
		"--format", "json",
		"--out", exportDir)
	if err != nil {
		t.Fatalf("export failed: %v\noutput: %s", err, out)
	}

	// verify exported files exist
	for _, f := range []string{"contacts.json", "activities.json", "events.json"} {
		if _, err := os.Stat(exportDir + "/" + f); os.IsNotExist(err) {
			t.Errorf("exported file %s not found", f)
		}
	}

	// verify contacts.json content
	data, err := os.ReadFile(exportDir + "/contacts.json")
	if err != nil {
		t.Fatalf("read contacts.json: %v", err)
	}
	var contacts []model.Contact
	if err := json.Unmarshal(data, &contacts); err != nil {
		t.Fatalf("parse exported contacts: %v", err)
	}
	if len(contacts) != 1 {
		t.Errorf("expected 1 contact, got %d", len(contacts))
	}
	if contacts[0].Name != "导出测试" {
		t.Errorf("unexpected name: %s", contacts[0].Name)
	}
}


func TestDoctor(t *testing.T) {
	dir := t.TempDir()
	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	out, err := executeCommand(t, "--data-dir", dir, "doctor")
	if err != nil {
		t.Fatalf("doctor failed: %v (out: %s)", err, out)
	}
	if !strings.Contains(out, "PASS") && !strings.Contains(out, "pass") {
		t.Errorf("doctor output unexpected: %s", out)
	}

	t.Run("with data", func(t *testing.T) {
		dir2 := t.TempDir()
		executeCommand(t, "--data-dir", dir2, "init")
		executeCommand(t, "--data-dir", dir2, "--format", "json",
			"contact", "upsert",
			"--name", "测试", "--email", "test@test.com", "--actor", "tester")
		out, err := executeCommand(t, "--data-dir", dir2, "doctor")
		if err != nil {
			t.Fatalf("doctor with data: %v (out: %s)", err, out)
		}
		if strings.Contains(out, "FAIL") {
			t.Errorf("unexpected issues: %s", out)
		}
	})

	t.Run("json output", func(t *testing.T) {
		dir3 := t.TempDir()
		executeCommand(t, "--data-dir", dir3, "init")
		out, err := executeCommand(t, "--data-dir", dir3, "--format", "json", "doctor")
		if err != nil {
			t.Fatalf("doctor json: %v (out: %s)", err, out)
		}
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("doctor json parse: %v (out: %s)", err, out)
		}
		if result["status"] != "ok" {
			t.Errorf("expected status ok, got: %v", result["status"])
		}
	})

	t.Run("fix flag", func(t *testing.T) {
		dir4 := t.TempDir()
		executeCommand(t, "--data-dir", dir4, "init")
		out, err := executeCommand(t, "--data-dir", dir4, "doctor", "--fix")
		if err != nil {
			t.Logf("doctor --fix output: %s", out)
		}
	})
}
