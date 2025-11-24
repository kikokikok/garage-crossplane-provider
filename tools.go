//go:build tools
// +build tools

// Package tools imports packages that are used for code generation
package tools

import (
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
