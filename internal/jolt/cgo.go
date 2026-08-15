//go:build windows

package jolt

/*
#cgo CXXFLAGS: -std=c++17 -I${SRCDIR}/../../third_party/jolt/JoltPhysics -DNDEBUG -DJPH_DISABLE_CUSTOM_ALLOCATOR -DJPH_PROFILE_ENABLED -DJPH_DEBUG_RENDERER -DJPH_OBJECT_STREAM
*/
import "C"
