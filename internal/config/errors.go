package config

import "errors"

var (
	ErrConfigVersionUnsupported = errors.New("unsupported mcpgen.json version")
	ErrServerNameRequired       = errors.New("server.name is required")
	ErrServerModuleInvalid      = errors.New("server.module is not a valid Go module path")
	ErrTransportTypeInvalid     = errors.New("transport.type must be \"stdio\" or \"http\"")
	ErrTransportPortInvalid     = errors.New("transport.httpPort must be between 1 and 65535")
	ErrProtocolInvalid          = errors.New("protocol must be \"compat\" or \"latest-only\"")
	ErrDuplicateID              = errors.New("duplicate id")
	ErrURIMissingScheme         = errors.New("uri must include a scheme (e.g. file:///, https://)")
	ErrConfigNotFound           = errors.New("mcpgen.json not found")
)
