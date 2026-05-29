/**
 * add_test_evidence_and_known_limits.js — PR 10 migration helper
 *
 * Purpose: Two modifications:
 *   (1) Insert "Test plan evidence requirement" section into CLAUDE.md
 *       — mandates actual command output in PRs, not empty checkmarks.
 *   (2) Insert "§13 Known Limitations" into docs/i18n-design.md
 *       — documents PR 8 regression, CJK fullwidth paren bug, and prevention.
 *
 * Input:  CLAUDE.md, docs/i18n-design.md
 * Output: Same files with new sections inserted.
 *
 * Usage:  node scripts/add_test_evidence_and_known_limits.js
 *
 * Status: One-shot script, used in PR 10. Kept for reference.
 *         Note: the CLAUDE.md section was later restructured from ### heading
 *         to nested bullet (PR 11). The script inserts the original form.
 */
const fs = require('fs');

// ============================================================
// 1. CLAUDE.md — Test plan evidence requirement
// ============================================================
{
  const filepath = 'CLAUDE.md';
  let c = fs.readFileSync(filepath, 'utf8');
  const crlf = c.includes('\r\n');
  if (crlf) c = c.replace(/\r\n/g, '\n');

  // Insert after the test bullet (line 129 in current file)
  // The marker: "- **测试是必需品不是 nice-to-have**。..."
  const testBullet = '- **测试是必需品不是 nice-to-have**。任何新增的 internal/ 代码都必须有对应的 _test.go';

  const evidenceSection = `
### Test plan evidence requirement

Each PR's Test plan must be backed by actual command output, not empty checkmarks:

- \`go vet ./... passes\` → paste the actual vet output
  (should be empty or "no issues found")
- \`go test ./... passes\` → paste the last 5-10 lines of test output
  (must show PASS/FAIL summary and package list)
- \`go build ./... succeeds\` → paste the build output
  (all packages listed without errors)

If a command fails or reports issues, the PR must:
1. Report the failure explicitly in the PR description
2. Fix it, or explain why it is a known issue
3. **Not** mark it as passing

Empty checkmarks without evidence will be treated as false reporting
and the PR will be returned for correction.
`;

  c = c.replace(testBullet, testBullet + evidenceSection);

  if (crlf) c = c.replace(/\n/g, '\r\n');
  fs.writeFileSync(filepath, c, 'utf8');
  console.log('Updated CLAUDE.md');
}

// ============================================================
// 2. docs/i18n-design.md — §13 Known Limitations
// ============================================================
{
  const filepath = 'docs/i18n-design.md';
  let c = fs.readFileSync(filepath, 'utf8');
  const crlf = c.includes('\r\n');
  if (crlf) c = c.replace(/\r\n/g, '\n');

  // The marker for the start of §13 Exclusions
  const exclusionHeader = '## 13. Exclusions (v1.0 scope)';

  const knownLimitations = `## 13. Known Limitations

### 13.1 Global i18n state leaks across tests

The i18n package uses global state via \`SetLang()\` + \`T()\`. Language
changes persist for the process lifetime, which means tests that change
language can affect subsequent tests in the same package.

**Mitigation:** any test that calls \`SetLang("zh")\` must
\`defer SetLang("en")\` to restore the default.

**Future direction (v0.2.0):** consider per-Localizer or context-based API
so each test or request gets its own language instance.

### 13.2 Lesson learned from PR 8 regression

PR 8 (alert+events+memory migration) introduced two real regressions:

1. **events.go compile error:** A migration script's string match broke at
   a Chinese fullwidth parenthesis (U+FF08), leaving "（多 Agent 协作）"
   outside the \`i18n.T()\` call. The package could not compile, but this
   was not caught because the Test plan checkmarks were filled without
   running the actual commands.

2. **TestT_Interpolation_Int always failing:** A test asserting plural form
   was written with \`T()\` instead of \`Tn()\`. The test had been failing
   since creation but was masked by issue #1 — when a package cannot
   compile, no test in it can run.

**Detection:** PR 9a's events.go re-migration accidentally fixed issue #1,
which immediately surfaced issue #2 on the first real test run.

**Prevention (now enforced in CLAUDE.md):**
- Migration scripts must handle CJK fullwidth characters in regex patterns
- Each PR commit must include actual test execution output as evidence
- Tests using \`T()\` with numeric arguments must verify the result
  matches the expected singular/plural form

---
`;

  // Insert known limitations before exclusions
  c = c.replace(exclusionHeader, knownLimitations + exclusionHeader);

  // Renumber §13 Exclusions → §14
  c = c.replace('## 13. Exclusions (v1.0 scope)', '## 14. Exclusions (v1.0 scope)');

  if (crlf) c = c.replace(/\n/g, '\r\n');
  fs.writeFileSync(filepath, c, 'utf8');
  console.log('Updated docs/i18n-design.md');
}
