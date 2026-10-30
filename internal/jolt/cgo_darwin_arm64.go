//go:build darwin && arm64

package jolt

/*
#cgo LDFLAGS: -L${SRCDIR}/../../third_party/jolt/darwin_arm64 -L${GOPATH}/pkg/mod/github.com/bbitechnologies/jolt-go@v0.8.4/jolt/lib/darwin_arm64 -ljolt_wrapper -lJolt -lc++
*/
import "C"
