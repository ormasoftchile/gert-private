// Package eval contains the conformance-facing scaffold for the GERT expression runtime.
// The package coordinates the GXL, GIS, and GCP subpackages without bypassing
// the parser/planner gate; concrete parsers and evaluators live in the grammar-
// specific packages and share the PJVM types from core.
package eval
