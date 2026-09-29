; Physical rope — gravity sag, collision, tension, and force transfer.
; Space swings the weight, R resets, Esc quits after a few frames.

SetWindowTitle("BitShin BASIC — Physical Rope")
Graphics3D(1000, 620, 0, 2)
SetCameraClsColor(54, 78, 108)
SetAmbientLight(80, 84, 92)

; Camera
cam = CreateCamera()
cam.Position([0, 6, -17])
CameraLookAt(cam, 0, 4, 8)

; Sun
sun = CreateDirectionalLight()
SetLightDirection(sun, 48, 34, 0)
SetLightColor(sun, 255, 238, 206)

; Static ground
ground = CreateCube().Scale(20, 0.3, 20).Position([0, 0, 8]).Color(68, 92, 62)
CreateRigidBodyBox(ground, 20, 0.3, 20, 0)

; Rope anchor (visual only)
anchor = CreateSphere(10).Scale(0.22, 0.22, 0.22).Position([0, 9, 8]).Color(245, 205, 72)

; Hanging weight
weight = CreateCube().Scale(0.7, 0.7, 0.7).Position([0, 3.5, 8]).Color(194, 62, 48)
CreateRigidBodyBox(weight, 0.7, 0.7, 0.7, 18)

; Anchored rope from world point to weight
rope = CreateRopeAnchored(0, weight, 0, 9, 8, 0, 0.7, 0, 7.2, 14, 0.045)
SetRopeColor(rope, 224, 194, 116)
SetRopeMass(rope, 0.12)
SetRopeDamping(rope, 0.2)
SetRopeStrength(rope, 1000, 220, 900)

frames = 0
While 1
    frames = frames + 1
    If frames > 8 And KeyHit(KEY_ESCAPE) Then Exit
    If KeyHit(KEY_SPACE) Then ApplyImpulse(weight, 7, 2, 0)
    If KeyHit(KEY_R) Then
        SetPosition(weight, 0, 3.5, 8)
        SetVelocity(weight, 0, 0, 0)
        ResetRope(rope)
    EndIf

    UpdateWorld
    RenderWorld
    Text(16, 16, "Physical Rope")
    Text(16, 40, "Space swing    R reset    Esc quit")
    Text(16, 64, "Tension " + Int(RopeTension(rope) * 100) + "%")
    Flip
Wend
End
