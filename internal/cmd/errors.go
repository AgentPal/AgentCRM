package cmd

import "errors"

// Sentinel errors for the cmd package.
// These stay English internally; userFacingError() translates them at the CLI boundary.
var (
	ErrContactNameRequired = errors.New("--name is required")
	ErrContactSetRequired  = errors.New("at least one --set is required")
	ErrContactFieldRequired = errors.New("--field is required")
)
