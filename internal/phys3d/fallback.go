//go:build nojolt || !(((linux && (amd64 || arm64)) || (darwin && arm64) || windows))

package phys3d

func New() World { return newFallback() }
