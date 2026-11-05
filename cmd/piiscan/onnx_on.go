//go:build onnx

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/techiek77-ui/poc1/pkg/pii/onnx"
	"github.com/techiek77-ui/poc1/pkg/pii/verify"
)

var (
	onnxDir = flag.String("onnx", "", "Tier B model directory (model.int8.onnx, tokenizer.json, manifest.json)")
	ortLib  = flag.String("ort", os.Getenv("PII_ORT_LIB"), "path to libonnxruntime.so / onnxruntime.dll / .dylib")
)

func extraVerifier() (verify.Verifier, func()) {
	if *onnxDir == "" {
		return nil, func() {}
	}
	if err := onnx.Init(*ortLib); err != nil {
		fmt.Fprintln(os.Stderr, "onnx runtime:", err)
		os.Exit(1)
	}
	v, err := onnx.New(*onnxDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "onnx model:", err)
		os.Exit(1)
	}
	return v, v.Close
}
