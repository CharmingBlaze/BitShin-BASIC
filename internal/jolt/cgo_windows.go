//go:build windows

package jolt

/*
#cgo LDFLAGS: -L${SRCDIR}/../../third_party/jolt/windows_amd64 -ljolt_wrapper -lJolt -lc++ -lm
*/
import "C"
