import json
import os

with open("internal/store/store_test.go", "r", encoding="utf-8") as f:
    content = f.read()

# Build the new tests
tests = """

func TestReindex(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	c := model.NewContact()
	c.ID = "cnt_01REINDEX"
	c.Name = "Reindex Test"
	c.Emails = []string{"reindex@test.com"}
	c.Slug = "reindex-test"
	if err := s.FS.WriteContact(c); err != nil {
		t.Fatalf("WriteContact: %v", err)
	}

	d := model.NewDeal()
	d.ID = "deal_01REINDEX"
	d.Title = "Reindex Deal"
	d.Slug = "reindex-deal"
	if err := s.FS.WriteDeal(d); err != nil {
		t.Fatalf("WriteDeal: %v", err)
	}

	if err := s.Reindex(); err != nil {
		t.Fatalf("Reindex: %v", err)
	}

	contact, err := s.Contacts.GetByID("cnt_01REINDEX")
	if err != nil {
		t.Errorf("GetByID after reindex: %v", err)
	}
	if contact.Name != "Reindex Test" {
		t.Errorf("expected Reindex Test, got %s", contact.Name)
	}

	deal, err := s.Deals.GetByID("deal_01REINDEX")
	if err != nil {
		t.Errorf("deal GetByID after reindex: %v", err)
	}
	if deal.Title != "Reindex Deal" {
		t.Errorf("expected Reindex Deal, got %s", deal.Title)
	}
}

func TestFileStore_UpdateContactField(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStore(dir)
	fs.Init()

	c := model.NewContact()
	c.ID = "cnt_01FIELD"
	c.Name = "Field Test"
	c.Company = "Old Corp"
	c.Title = "Engineer"
	c.Slug = "field-test"
	if err := fs.WriteContact(c); err != nil {
		t.Fatalf("WriteContact: %v", err)
	}

	oldVal, err := fs.UpdateContactField("field-test", "company", "New Corp", "rebrand", "tester")
	if err != nil {
		t.Fatalf("UpdateContactField: %v", err)
	}
	if oldVal != "Old Corp" {
		t.Errorf("expected old=Old Corp, got %s", oldVal)
	}

	updated, err := fs.ReadContact("field-test")
	if err != nil {
		t.Fatalf("ReadContact: %v", err)
	}
	if updated.Company != "New Corp" {
		t.Errorf("expected New Corp, got %s", updated.Company)
	}
	if len(updated.CompanyHistory) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(updated.CompanyHistory))
	}
	if updated.CompanyHistory[0].Value != "Old Corp" {
		t.Errorf("expected history value Old Corp, got %s", updated.CompanyHistory[0].Value)
	}

	// Same value update should be no-op
	oldVal2, err := fs.UpdateContactField("field-test", "company", "New Corp", "", "tester")
	if err != nil {
		t.Fatalf("UpdateContactField same value: %v", err)
	}
	if oldVal2 != "New Corp" {
		t.Errorf("expected old=New Corp for no-op, got %s", oldVal2)
	}

	// Update title
	_, err = fs.UpdateContactField("field-test", "title", "Senior Engineer", "promotion", "tester")
	if err != nil {
		t.Fatalf("UpdateContactField title: %v", err)
	}
	updated, _ = fs.ReadContact("field-test")
	if updated.Title != "Senior Engineer" {
		t.Errorf("expected Senior Engineer, got %s", updated.Title)
	}
	if len(updated.TitleHistory) != 1 {
		t.Errorf("expected 1 title history entry, got %d", len(updated.TitleHistory))
	}
}

func TestFileStore_UpdateDealField(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStore(dir)
	fs.Init()

	d := model.NewDeal()
	d.ID = "deal_01UPDFIELD"
	d.Title = "Update Field Deal"
	d.Slug = "update-field-deal"
	d.Amount = 10000
	d.Stage = "lead"
	if err := fs.WriteDeal(d); err != nil {
		t.Fatalf("WriteDeal: %v", err)
	}

	if err := fs.UpdateDealField("update-field-deal", "stage", "qualified", "lead", "progressing", "tester"); err != nil {
		t.Fatalf("UpdateDealField stage: %v", err)
	}
	updated, _ := fs.ReadDeal("update-field-deal")
	if updated.Stage != "qualified" {
		t.Errorf("expected qualified, got %s", updated.Stage)
	}
	if len(updated.StageHistory) != 1 {
		t.Errorf("expected 1 stage history, got %d", len(updated.StageHistory))
	}

	if err := fs.UpdateDealField("update-field-deal", "amount", "15000", "10000", "expanded scope", "tester"); err != nil {
		t.Fatalf("UpdateDealField amount: %v", err)
	}
	updated, _ = fs.ReadDeal("update-field-deal")
	if updated.Amount != 15000 {
		t.Errorf("expected 15000, got %d", updated.Amount)
	}
}

func TestFileStore_MergeContactFiles(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStore(dir)
	fs.Init()

	keeper := model.NewContact()
	keeper.ID = "cnt_01KEEPER"
	keeper.Name = "Keeper"
	keeper.Company = "Keeper Co"
	keeper.Title = "CEO"
	keeper.Emails = []string{"keeper@co.com"}
	keeper.Tags = []string{"vip"}
	keeper.Slug = "keeper"
	if err := fs.WriteContact(keeper); err != nil {
		t.Fatalf("WriteContact keeper: %v", err)
	}

	mergee := model.NewContact()
	mergee.ID = "cnt_01MERGEE"
	mergee.Name = "M. Keeper"
	mergee.Company = "Keeper Co"
	mergee.Phones = []string{"+8613800000000"}
	mergee.Tags = []string{"partner"}
	mergee.Body = "Additional notes"
	mergee.Slug = "m-keeper"
	if err := fs.WriteContact(mergee); err != nil {
		t.Fatalf("WriteContact mergee: %v", err)
	}

	if err := fs.MergeContactFiles("cnt_01KEEPER", "cnt_01MERGEE", "keeper", "m-keeper", "duplicate", "tester"); err != nil {
		t.Fatalf("MergeContactFiles: %v", err)
	}

	mergedKeeper, err := fs.ReadContact("keeper")
	if err != nil {
		t.Fatalf("ReadContact keeper: %v", err)
	}
	if len(mergedKeeper.Phones) != 1 || mergedKeeper.Phones[0] != "+8613800000000" {
		t.Errorf("expected phone from mergee, got %v", mergedKeeper.Phones)
	}
	if len(mergedKeeper.Tags) != 2 {
		t.Errorf("expected 2 tags after merge, got %d: %v", len(mergedKeeper.Tags), mergedKeeper.Tags)
	}
	if mergedKeeper.Body != "Additional notes" {
		t.Errorf("expected body from mergee, got %s", mergedKeeper.Body)
	}

	_, err = fs.ReadContact("m-keeper")
	if err == nil {
		t.Errorf("expected mergee file to be gone")
	}
}

func TestMemos_Search(t *testing.T) {
	db := openTestDB(t)
	s := NewMemoStore(db, t.TempDir())

	s.Insert(&model.Memo{ID: "memo_01SRCH", ScopeType: "contact", ScopeID: "cnt_01", Text: "Prefers email communication"})
	s.Insert(&model.Memo{ID: "memo_02SRCH", ScopeType: "contact", ScopeID: "cnt_02", Text: "Decision maker is the CTO"})

	results, err := s.Search("email", "", "", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results")
	}
	foundEmail := false
	for _, r := range results {
		if r.Text == "Prefers email communication" {
			foundEmail = true
		}
	}
	if !foundEmail {
		t.Errorf("expected to find email memo in results")
	}

	scoped, err := s.Search("email", "contact", "cnt_01", 10)
	if err != nil {
		t.Fatalf("Scoped search: %v", err)
	}
	if len(scoped) == 0 {
		t.Error("expected scoped search results")
	}

	noResults, err := s.Search("nonexistenttermzzz", "", "", 10)
	if err != nil {
		t.Fatalf("No-results search: %v", err)
	}
	if len(noResults) != 0 {
		t.Errorf("expected 0 results, got %d", len(noResults))
	}
}

func TestMemos_Commit(t *testing.T) {
	dir := t.TempDir()
	db := openTestDB(t)
	s := NewMemoStore(db, dir)

	proposalDir := dir + "/proposals"
	os.MkdirAll(proposalDir, 0755)
	proposal := model.Proposal{
		ID: "prop_01COMMIT", Timestamp: "2026-01-01T00:00:00Z",
		Actor: "tester", Scope: "contact:cnt_01",
		Statement: "New fact", Status: "clean",
	}
	data, _ := json.Marshal(proposal)
	os.WriteFile(proposalDir+"/pending.jsonl", append(data, '\n'), 0644)

	s.Insert(&model.Memo{ID: "memo_01SUPERSEDE", ScopeType: "contact", ScopeID: "cnt_01", Text: "Old fact"})

	if err := s.Commit("prop_01COMMIT", "supersede"); err != nil {
		t.Fatalf("Commit supersede: %v", err)
	}

	oldMemo, err := s.GetByID("memo_01SUPERSEDE")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if !oldMemo.Expired {
		t.Error("expected old memo to be expired after supersede")
	}

	// Test keep-both
	proposal2 := model.Proposal{
		ID: "prop_02COMMIT", Timestamp: "2026-01-01T00:00:00Z",
		Actor: "tester", Scope: "contact:cnt_01",
		Statement: "Another fact", Status: "clean",
	}
	data2, _ := json.Marshal(proposal2)
	os.WriteFile(proposalDir+"/pending.jsonl", append(data2, '\n'), 0644)
	if err := s.Commit("prop_02COMMIT", "keep-both"); err != nil {
		t.Fatalf("Commit keep-both: %v", err)
	}

	// Test reject
	proposal3 := model.Proposal{
		ID: "prop_03COMMIT", Timestamp: "2026-01-01T00:00:00Z",
		Actor: "tester", Scope: "contact:cnt_01",
		Statement: "Rejected fact", Status: "clean",
	}
	data3, _ := json.Marshal(proposal3)
	os.WriteFile(proposalDir+"/pending.jsonl", append(data3, '\n'), 0644)
	if err := s.Commit("prop_03COMMIT", "reject"); err != nil {
		t.Fatalf("Commit reject: %v", err)
	}
}

func TestFileStore_RootDir(t *testing.T) {
	fs := NewFileStore("/tmp/test-root")
	if fs.RootDir() != "/tmp/test-root" {
		t.Errorf("expected /tmp/test-root, got %s", fs.RootDir())
	}
}

func TestMemPropose_Conflict(t *testing.T) {
	db := openTestDB(t)
	s := NewMemoStore(db, t.TempDir())

	s.Insert(&model.Memo{ID: "memo_01CONF", ScopeType: "contact", ScopeID: "cnt_01", Text: "Customer prefers email"})

	p, err := s.Propose("contact:cnt_01", "Customer prefers email communication", "from email", "tester", 0.9)
	if err != nil {
		t.Fatalf("Propose: %v", err)
	}
	if p.Status != "conflict" {
		t.Errorf("expected conflict, got %s", p.Status)
	}
	if len(p.ConflictWith) == 0 {
		t.Errorf("expected conflict_with entries")
	}
}
"""

content += tests

with open("internal/store/store_test.go", "w", encoding="utf-8") as f:
    f.write(content)
print("OK: added %d bytes of tests" % len(tests))
