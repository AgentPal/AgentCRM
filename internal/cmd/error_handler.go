package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/AgentPal/AgentCRM/internal/i18n"
)

// userFacingError translates sentinel errors to the user's configured language.
// Unrecognized errors pass through unchanged (English fallback).
func userFacingError(err error) string {
	switch {
	case errors.Is(err, ErrContactNameRequired):
		return i18n.T("error.contact.name.required")
	case errors.Is(err, ErrContactSetRequired):
		return i18n.T("error.contact.set.required")
	case errors.Is(err, ErrContactFieldRequired):
		return i18n.T("error.contact.field.required")
	case errors.Is(err, ErrDealTitleRequired):
		return i18n.T("error.deal.title.required")
	case errors.Is(err, ErrDealSetRequired):
		return i18n.T("error.deal.set.required")
	case errors.Is(err, ErrDealStageIrreversible):
		return i18n.T("error.deal.stage.irreversible")
	case errors.Is(err, ErrDealFieldRequired):
		return i18n.T("error.deal.field.required")
	case errors.Is(err, ErrDealFieldUnsupported):
		return i18n.T("error.deal.field.unsupported")
	case errors.Is(err, ErrActivityContactRequired):
		return i18n.T("error.activity.contact.required")
	case errors.Is(err, ErrActivitySummaryRequired):
		return i18n.T("error.activity.summary.required")
	case errors.Is(err, ErrActivityDedupeKeyRequired):
		return i18n.T("error.activity.dedupe_key.required")
	case errors.Is(err, ErrMemoryScopeTextRequired):
		return i18n.T("error.memory.scope_text.required")
	case errors.Is(err, ErrMemoryScopeRequired):
		return i18n.T("error.memory.scope.required")
	case errors.Is(err, ErrMemoryScopeStatementRequired):
		return i18n.T("error.memory.scope_statement.required")
	case errors.Is(err, ErrMemoryProposalActionRequired):
		return i18n.T("error.memory.proposal_action.required")
	default:
		return err.Error()
	}
}

// Execute is the main entry point. It runs the root command and handles error
// display with i18n translation. Errors are printed to stderr; normal output to stdout.
func Execute() {
	i18n.MustInit("")
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, userFacingError(err))
		os.Exit(1)
	}
}
