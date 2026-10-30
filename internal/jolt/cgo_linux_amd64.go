//go:build linux && amd64

package jolt

/*
#cgo LDFLAGS: -L${SRCDIR}/../../third_party/jolt/linux_amd64 -L${GOPATH}/pkg/mod/github.com/bbitechnologies/jolt-go@v0.8.4/jolt/lib/linux_amd64 -ljolt_wrapper -lJolt -lstdc++ -lm -lpthread
*/
import "C"
