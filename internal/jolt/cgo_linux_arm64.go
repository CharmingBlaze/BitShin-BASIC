//go:build linux && arm64

package jolt

/*
#cgo LDFLAGS: -L${SRCDIR}/../../third_party/jolt/linux_arm64 -L${GOPATH}/pkg/mod/github.com/bbitechnologies/jolt-go@v0.8.4/jolt/lib/linux_arm64 -ljolt_wrapper -lJolt -lstdc++ -lm -lpthread
*/
import "C"
