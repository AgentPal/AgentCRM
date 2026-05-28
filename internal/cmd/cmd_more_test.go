package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/AgentPal/AgentCRM/internal/model"
)

func TestExportCSV(t *testing.T) {
	dir := t.TempDir()

	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Create a contact
	_, err = executeCommand(t, "--data-dir", dir, "--format", "json",
		"contact", "upsert",
		"--name", "CSV导出测试",
		"--email", "csv@example.com",
		"--company", "CSVCorp")
	if err != nil {
		t.Fatalf("create contact failed: %v", err)
	}

	exportDir := dir + "/csv-export"
	out, err := executeCommand(t, "--data-dir", dir,
		"export",
		"--format", "csv",
		"--out", exportDir)
	if err != nil {
		t.Fatalf("export csv failed: %v\noutput: %s", err, out)
	}

	// Verify exported CSV files exist
	for _, f := range []string{"contacts.csv"} {
		path := exportDir + "/" + f
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("exported file %s not found", f)
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if len(data) == 0 {
			t.Errorf("%s is empty", f)
		}
	}

	// Verify contacts.csv content
	data, err := os.ReadFile(exportDir + "/contacts.csv")
	if err != nil {
		t.Fatalf("read contacts.csv: %v", err)
	}
	if !strings.Contains(string(data), "CSV导出测试") {
		t.Errorf("expected contact name in CSV, got: %s", data)
	}
}

func TestImportVCard(t *testing.T) {
	dir := t.TempDir()

	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Create a vCard file
	vcardPath := dir + "/test.vcf"
	vcardContent := `BEGIN:VCARD
VERSION:3.0
FN:张三
EMAIL:zhangsan@import.com
TEL:+8613800000001
ORG:ImportCorp
TITLE:Manager
NOTE:Imported from test
END:VCARD`
	if err := os.WriteFile(vcardPath, []byte(vcardContent), 0644); err != nil {
		t.Fatalf("write vcard: %v", err)
	}

	out, err := executeCommand(t, "--data-dir", dir,
		"import",
		"--source", "vcard",
		"--file", vcardPath)
	if err != nil {
		t.Fatalf("import vcard failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "已导入") {
		t.Errorf("unexpected output: %s", out)
	}

	// Verify contact was imported
	listOut, err := executeCommand(t, "--data-dir", dir, "--format", "json",
		"contact", "list")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	var results []model.ContactSearchResult
	if err := json.Unmarshal([]byte(listOut), &results); err != nil {
		t.Fatalf("parse list: %v\nraw: %s", err, listOut)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 imported contact")
	}
}

func TestImportCSV(t *testing.T) {
	dir := t.TempDir()

	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Create a CSV file
	csvPath := dir + "/test.csv"
	csvContent := "name,email,company,title,phone,tags,source\n李四,lisi@import.com,CSVCorp,Engineer,13900000001,dev;partner,test\n王五,wangwu@import.com,OtherCorp,Designer,,,test"
	if err := os.WriteFile(csvPath, []byte(csvContent), 0644); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	out, err := executeCommand(t, "--data-dir", dir,
		"import",
		"--source", "csv",
		"--file", csvPath)
	if err != nil {
		t.Fatalf("import csv failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "已导入") {
		t.Errorf("unexpected output: %s", out)
	}

	// Verify contacts were imported
	listOut, err := executeCommand(t, "--data-dir", dir, "--format", "json",
		"contact", "list")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	var results []model.ContactSearchResult
	if err := json.Unmarshal([]byte(listOut), &results); err != nil {
		t.Fatalf("parse list: %v\nraw: %s", err, listOut)
	}
	if len(results) < 2 {
		t.Errorf("expected at least 2 imported contacts, got %d", len(results))
	}
}

func TestExport_UnsupportedFormat(t *testing.T) {
	dir := t.TempDir()

	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	_, err = executeCommand(t, "--data-dir", dir, "export", "--format", "xml", "--out", dir+"/out")
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestImport_Errors(t *testing.T) {
	dir := t.TempDir()

	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// No --file flag on import should error
	_, err = executeCommand(t, "--data-dir", dir, "import", "--source", "csv")
	if err == nil {
		t.Error("expected error for missing --file")
	}

	// Unsupported source
	tmpFile := dir + "/test.txt"
	os.WriteFile(tmpFile, []byte("x"), 0644)
	_, err = executeCommand(t, "--data-dir", dir, "import", "--source", "invalid", "--file", tmpFile)
	if err == nil {
		t.Error("expected error for unsupported source")
	}
}

func TestExportJSON_WithDeal(t *testing.T) {
	dir := t.TempDir()

	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Create contact
	contactOut, err := executeCommand(t, "--data-dir", dir, "--format", "json",
		"contact", "upsert",
		"--name", "导出测试含商机",
		"--email", "dealdeal@test.com")
	if err != nil {
		t.Fatalf("create contact failed: %v", err)
	}
	var cRes struct{ ID string }
	json.Unmarshal([]byte(contactOut), &cRes)

	// Create deal (exercises readDealFile, parseDealMarkdown)
	_, err = executeCommand(t, "--data-dir", dir,
		"deal", "create",
		"--title", "测试商机",
		"--contact", cRes.ID,
		"--amount", "88888")
	if err != nil {
		t.Fatalf("create deal failed: %v", err)
	}

	// Export JSON
	exportDir := dir + "/json-export"
	out, err := executeCommand(t, "--data-dir", dir,
		"export", "--format", "json", "--out", exportDir)
	if err != nil {
		t.Fatalf("export json failed: %v\noutput: %s", err, out)
	}

	// Verify deals.json exists
	data, err := os.ReadFile(exportDir + "/deals.json")
	if err != nil {
		t.Fatalf("read deals.json: %v", err)
	}
	var deals []model.Deal
	if err := json.Unmarshal(data, &deals); err != nil {
		t.Fatalf("parse deals: %v\nraw: %s", err, string(data))
	}
	if len(deals) != 1 {
		t.Errorf("expected 1 deal, got %d", len(deals))
	}
	if deals[0].Title != "测试商机" {
		t.Errorf("unexpected deal title: %s", deals[0].Title)
	}
}

func TestEvents_Subscribers(t *testing.T) {
	dir := t.TempDir()

	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Subscribers list (no subscribers yet)
	out, err := executeCommand(t, "--data-dir", dir, "events", "subscribers", "list")
	if err != nil {
		t.Fatalf("subscribers list failed: %v\noutput: %s", err, out)
	}

	// Reset a subscriber
	out, err = executeCommand(t, "--data-dir", dir, "events", "subscribers", "reset", "test-actor")
	if err != nil {
		t.Fatalf("subscribers reset failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "已重置") {
		t.Errorf("unexpected reset output: %s", out)
	}
}

func TestEvents_SubscribersListJSON(t *testing.T) {
	dir := t.TempDir()

	_, err := executeCommand(t, "--data-dir", dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Subscribers list in JSON format
	out, err := executeCommand(t, "--data-dir", dir, "--format", "json",
		"events", "subscribers", "list")
	if err != nil {
		t.Fatalf("subscribers list json failed: %v\noutput: %s", err, out)
	}
	var subs []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &subs); err != nil {
		t.Fatalf("parse subscribers json: %v\nraw: %s", err, out)
	}
}
