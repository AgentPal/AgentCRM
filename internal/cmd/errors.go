package cmd

import "errors"

// Sentinel errors for the cmd package.
// These stay English internally; userFacingError() translates them at the CLI boundary.
var (
	// Contact (PR 6)
	ErrContactNameRequired  = errors.New("--name is required")
	ErrContactSetRequired   = errors.New("at least one --set is required")
	ErrContactFieldRequired = errors.New("--field is required")

	// Deal (PR 7)
	ErrDealTitleRequired      = errors.New("--title is required")
	ErrDealSetRequired        = errors.New("at least one --set is required")
	ErrDealStageIrreversible  = errors.New("cannot transition from %s to %s: won/lost is irreversible")
	ErrDealFieldRequired      = errors.New("--field is required")
	ErrDealFieldUnsupported   = errors.New("unsupported field: %s (supported: stage, amount)")

	// Activity (PR 7)
	ErrActivityContactRequired  = errors.New("--contact is required")
	ErrActivitySummaryRequired  = errors.New("--summary is required")
	ErrActivityDedupeKeyRequired = errors.New("--dedupe-key is required")

	// Memory (PR 8)
	ErrMemoryScopeTextRequired      = errors.New("--scope and --text are required")
	ErrMemoryScopeRequired          = errors.New("--scope is required")
	ErrMemoryScopeStatementRequired = errors.New("--scope and --statement are required")
	ErrMemoryProposalActionRequired = errors.New("--proposal-id and --action are required")
)
