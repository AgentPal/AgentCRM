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
	default:
		return err.Error()
	}
}

// Execute is the main entry point. It runs the root command and handles error
// display with i18n translation. Errors are printed to stderr; normal output to stdout.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, userFacingError(err))
		os.Exit(1)
	}
}
