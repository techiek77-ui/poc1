//go:build !onnx

package main

import "github.com/techiek77-ui/poc1/pkg/pii/verify"

// extraVerifier is a no-op without the onnx build tag.
func extraVerifier() (verify.Verifier, func()) { return nil, func() {} }
