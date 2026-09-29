; Math Helpers — trig, distance, easing, and vector helpers (console only)

Print("BitShin BASIC math")

; Trig and distance (angles in degrees)
Print("trig", Int(Sin(90)), Int(Cos(0)), Int(ATan2(1, 0)))
Print("dist", Dist(0, 0, 3, 4), Distance3D(0, 0, 0, 0, 0, 5))

; Clamp, lerp, and easing
Print("clamp/lerp", Clamp(15, 0, 10), Lerp(0, 10, 0.5), InvLerp(0, 10, 5))
Print("ease", EaseIn(0.5), EaseOut(0.5), SmoothStep(0, 1, 0.5))

; Angles and approach
Print("angles", WrapAngle(270), AngleDelta(10, 350))
Print("approach", Approach(0, 10, 3), ApproachAngle(170, -170, 10))

; Direction and movement points
Print("dir", DirX(90), DirZ(0), MovePointX(0, 90, 10), MovePointZ(0, 0, 10))
Print("norm", NormX(3, 4), NormY(3, 4))
Print("point", PointYaw(0, 0, 10, 0), PointPitch(0, 0, 0, 0, 1, 0))

; Dot, cross, reflect, rotate
Print("dot/cross", Dot2D(1, 0, 0, 1), CrossZ(1, 0, 0, 0, 1, 0))
Print("reflect", ReflectX(1, -1, 0, 1), ReflectY(1, -1, 0, 1))
Print("rotate", RotatedX(1, 0, 90), RotatedY(1, 0, 90))

; Deterministic random (same min/max)
Print("rnd", Rand(1, 1))
End
