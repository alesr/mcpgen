package config

import "errors"

var (
	ErrServerNameRequired   = errors.New("server name is required")
	ErrServerModuleInvalid  = errors.New("server module is not a valid module path")
	ErrTransportTypeInvalid = errors.New("transport type is invalid")
	ErrTransportPortInvalid = errors.New("transport port is out of range")
	ErrURIMissingScheme     = errors.New("uri is missing scheme")
)
