// Package gobatfish is a Go client library for Batfish, mirroring the public
// API of pybatfish (https://github.com/batfish/pybatfish).
//
// The package name is intentionally gobatfish rather than pybatfish; the
// subpackages map onto pybatfish modules:
//
//	pybatfish                -> gobatfish            (version metadata)
//	pybatfish.client         -> gobatfish/client
//	pybatfish.datamodel      -> gobatfish/datamodel
//	pybatfish.datamodel.answer -> gobatfish/datamodel/answer
//	pybatfish.question       -> gobatfish/question
//	pybatfish.exception      -> gobatfish/exception
//	pybatfish.util           -> gobatfish/util
//
// pandas.DataFrame, which pybatfish exposes through TableAnswer.frame(), has no
// Go equivalent, so this library ships gobatfish/dataframe, a small, dependency
// free, object-typed data frame with the subset of pandas semantics that
// pybatfish relies on.
package gobatfish

// Version metadata, mirroring pybatfish/__init__.py.
const (
	// Desc is a human readable description of the library.
	Desc = "Go API and utilities for Batfish"
	// Name is the library name.
	Name = "gobatfish"
	// URL is the canonical project URL.
	URL = "https://github.com/81ueman/gobatfish"
	// Version is the library version, kept in sync with pybatfish.
	Version = "0.36.0"
)
