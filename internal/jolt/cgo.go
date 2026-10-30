//go:build windows || (linux && (amd64 || arm64)) || (darwin && arm64)

// Linux and macOS link libJolt from github.com/bbitechnologies/jolt-go v0.8.4
// (see cgo_linux_*.go and cgo_darwin_arm64.go). Keep that version in go.mod in sync.
// The default module cache is $GOPATH/pkg/mod. A custom GOMODCACHE needs the same
// libraries on the link path.

package jolt

/*
#cgo CXXFLAGS: -std=c++17 -I${SRCDIR}/../../third_party/jolt/JoltPhysics -DNDEBUG -DJPH_DISABLE_CUSTOM_ALLOCATOR -DJPH_PROFILE_ENABLED -DJPH_DEBUG_RENDERER -DJPH_OBJECT_STREAM
*/
import "C"
