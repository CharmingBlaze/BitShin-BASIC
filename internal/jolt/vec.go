//go:build windows

package jolt

import "math"

// Vec3 represents a 3D vector
type Vec3 struct {
	X, Y, Z float32
}

// NewVec3 constructs a vector.
func NewVec3(x, y, z float32) Vec3 {
	return Vec3{X: x, Y: y, Z: z}
}

// Add returns the sum of this vector and another vector
func (v Vec3) Add(other Vec3) Vec3 {
	return Vec3{X: v.X + other.X, Y: v.Y + other.Y, Z: v.Z + other.Z}
}

// Sub returns the difference of this vector and another vector
func (v Vec3) Sub(other Vec3) Vec3 {
	return Vec3{X: v.X - other.X, Y: v.Y - other.Y, Z: v.Z - other.Z}
}

// Mul returns this vector multiplied by a scalar
func (v Vec3) Mul(scalar float32) Vec3 {
	return Vec3{X: v.X * scalar, Y: v.Y * scalar, Z: v.Z * scalar}
}

// Dot returns the dot product of this vector with another vector
func (v Vec3) Dot(other Vec3) float32 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

// Length returns the magnitude (length) of the vector
func (v Vec3) Length() float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z)))
}

// Normalize returns a unit vector in the same direction as this vector.
// Returns a zero vector if the input vector has zero length.
func (v Vec3) Normalize() Vec3 {
	length := v.Length()
	if length == 0 {
		return Vec3{X: 0, Y: 0, Z: 0}
	}
	return Vec3{X: v.X / length, Y: v.Y / length, Z: v.Z / length}
}

// Quat represents a quaternion for rotations
type Quat struct {
	X, Y, Z, W float32
}

// Identity returns an identity quaternion (no rotation)
func QuatIdentity() Quat {
	return Quat{X: 0, Y: 0, Z: 0, W: 1}
}

// Rotate applies this quaternion to a vector (local → world).
func (q Quat) Rotate(v Vec3) Vec3 {
	ix := q.W*v.X + q.Y*v.Z - q.Z*v.Y
	iy := q.W*v.Y + q.Z*v.X - q.X*v.Z
	iz := q.W*v.Z + q.X*v.Y - q.Y*v.X
	iw := -q.X*v.X - q.Y*v.Y - q.Z*v.Z
	return Vec3{
		X: ix*q.W + iw*-q.X + iy*-q.Z - iz*-q.Y,
		Y: iy*q.W + iw*-q.Y + iz*-q.X - ix*-q.Z,
		Z: iz*q.W + iw*-q.Z + ix*-q.Y - iy*-q.X,
	}
}
