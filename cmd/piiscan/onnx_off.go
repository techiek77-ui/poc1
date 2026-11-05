//go:build !onnx

package main

import "github.com/techiek77-ui/poc1/pkg/pii/verify"

func extraVerifier() (verify.Verifier, func()) { return nil, func() {} }
