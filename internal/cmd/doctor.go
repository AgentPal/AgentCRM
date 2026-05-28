package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AgentPal/AgentCRM/internal/store"
	"github.com/spf13/cobra"
)

func init() {
	doctorCmd.Flags().Bool("fix", false, "auto-fix repairable issues")
	doctorCmd.Flags().String("check", "all", "checks: files,orphans,dedupe,memos")
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "check data integrity and health",
	RunE: func(cmd *cobra.Command, args []string) error {
		fix, _ := cmd.Flags().GetBool("fix")
		checkFlag, _ := cmd.Flags().GetString("check")
		isJSON := format == "json"

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		checks := map[string]func(*store.Store, bool) ([]string, error){
			"files":   checkFiles,
			"orphans": checkOrphans,
			"dedupe":  checkDedupeKeys,
			"memos":   checkMemos,
		}

		selected := []string{"files", "orphans", "dedupe", "memos"}
		if checkFlag != "" && checkFlag != "all" {
			selected = strings.Split(checkFlag, ",")
		}

		totalIssues := 0

		if !isJSON {
			fmt.Println("AgentCRM data integrity check")
			fmt.Println("==============================")
			fmt.Printf("  Data dir: %s\n", s.ConfigDir())
			fmt.Printf("  Database: %s\n", filepath.Join(s.ConfigDir(), ".index.db"))
		}

		for _, name := range selected {
			fn, ok := checks[name]
			if !ok {
				totalIssues++
				if !isJSON {
					fmt.Printf("? unknown check: %s\n", name)
				}
				continue
			}
			issues, err := fn(s, fix)
			if err != nil {
				totalIssues++
				if !isJSON {
					fmt.Printf("? %s: %v\n", name, err)
				}
				continue
			}
			if !isJSON {
				if len(issues) == 0 {
					fmt.Printf("OK %s: pass\n", name)
				} else {
					fmt.Printf("!! %s: %d issues\n", name, len(issues))
					for _, issue := range issues {
						fmt.Printf("  - %s\n", issue)
					}
				}
			}
			totalIssues += len(issues)
		}

		if isJSON {
			json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"status": "ok", "total_issues": totalIssues,
			})
		} else if totalIssues == 0 {
			fmt.Println()
			fmt.Println("PASS: all checks passed")
		} else {
			fmt.Println()
			fmt.Printf("FAIL: %d issues (use --fix to auto-repair)\n", totalIssues)
		}
		return nil
	},
}

func checkFiles(s *store.Store, fix bool) ([]string, error) {
	var issues []string

	contactFiles, err := s.FS.ListContactFiles()
	if err != nil {
		return nil, err
	}

	for _, slug := range contactFiles {
		slugOnly := strings.TrimSuffix(filepath.Base(slug), ".md")
		var count int
		err := s.DB.QueryRow(`SELECT COUNT(*) FROM contacts WHERE slug = ?`, slugOnly).Scan(&count)
		if err == nil && count == 0 {
			issues = append(issues, fmt.Sprintf("file exists but no DB index: %s", slug))
		}
	}

	// Check for merged-into file leftovers
	contactDir := filepath.Join(s.ConfigDir(), "contacts")
	entries, err := os.ReadDir(contactDir)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.Contains(e.Name(), ".merged-into-") {
				issues = append(issues, fmt.Sprintf("merged contact file leftover: %s", e.Name()))
			}
		}
	}

	return issues, nil
}

func checkOrphans(s *store.Store, fix bool) ([]string, error) {
	var issues []string

	// Orphaned activities
	rows, err := s.DB.Query(`SELECT id, contact_ids FROM activities WHERE contact_ids != '[]' AND contact_ids != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, cids string
		if err := rows.Scan(&id, &cids); err != nil {
			continue
		}
		var ids []string
		json.Unmarshal([]byte(cids), &ids)
		for _, cid := range ids {
			if _, err := s.Contacts.GetByID(cid); err != nil {
				issues = append(issues, fmt.Sprintf("activity %s references missing contact: %s", id, cid))
			}
		}
	}
	rows.Close()

	// Orphaned deals
	drows, err := s.DB.Query(`SELECT id, contact_ids FROM deals WHERE contact_ids != '[]' AND contact_ids != ''`)
	if err != nil {
		return nil, err
	}
	defer drows.Close()
	for drows.Next() {
		var id, cids string
		if err := drows.Scan(&id, &cids); err != nil {
			continue
		}
		var ids []string
		json.Unmarshal([]byte(cids), &ids)
		for _, cid := range ids {
			if _, err := s.Contacts.GetByID(cid); err != nil {
				issues = append(issues, fmt.Sprintf("deal %s references missing contact: %s", id, cid))
			}
		}
	}

	return issues, nil
}

func checkDedupeKeys(s *store.Store, fix bool) ([]string, error) {
	rows, err := s.DB.Query(`SELECT id, dedupe_key, summary FROM activities WHERE dedupe_key IS NULL OR dedupe_key = '' LIMIT 20`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []string
	for rows.Next() {
		var id, dk, summary string
		if err := rows.Scan(&id, &dk, &summary); err != nil {
			continue
		}
		issues = append(issues, fmt.Sprintf("activity %s missing dedupe_key: %s", id, summary))
	}
	if len(issues) > 0 {
		issues = append(issues, fmt.Sprintf("total %d activities missing dedupe keys", len(issues)))
	}
	return issues, nil
}

func checkMemos(s *store.Store, fix bool) ([]string, error) {
	memos, err := s.Memos.ListByScope("", "")
	if err != nil {
		return nil, err
	}

	var issues []string
	for _, m := range memos {
		if m.Expired {
			continue
		}
		if m.IsExpired("") {
			if fix {
				s.Memos.MarkExpired(m.ID)
			}
			issues = append(issues, fmt.Sprintf("memo %s expired: %s", m.ID, m.Text))
		}
	}
	if len(issues) > 0 && !fix {
		issues = append(issues, "run --fix to mark expired memos")
	}
	return issues, nil
}
