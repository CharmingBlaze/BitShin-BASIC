; Physics pile — 80 spheres spawn, then OptimizePhysics rebuilds the broadphase.
; Space kicks the first ball. Esc or X quits after a few frames.

SetWindowTitle("BitShin BASIC — Physics pile")
Graphics3D(960, 600, 0, 2)
SetCameraClsColor(18, 20, 28)

; Camera
cam = CreateCamera()
cam.Position([0, 10, -16])
cam.Rotate(22, 0, 0)
CreateLight()

; Static ground
ground = CreateCube().Scale(10, 0.2, 10).Position([0, 0, 8]).Color(50, 56, 70)
CreateRigidBodyBox(ground, 10, 0.2, 10, 0)

; Spawn burst (one-at-a-time bodies deepen Jolt's tree — OptimizePhysics rebuilds it)
first = 0
For i = 0 To 79
    b = CreateSphere(8).Scale(0.28, 0.28, 0.28).Position([Rnd(6) - 3, 3 + i * 0.22, 8 + Rnd(6) - 3]).Color(70 + Rand(120), 140 + Rand(80), 220)
    CreateBodySphere(b, 0.28, 1)
    If i = 0 Then first = b
Next
OptimizePhysics()

Print("Physics backend:", GetPhysicsBackend())

frames = 0
While 1
    frames = frames + 1
    If KeyHit(KEY_SPACE) Then ApplyImpulse(first, 0, 10, 4)
    UpdateWorld
    RenderWorld
    Text(16, 16, "Physics pile  |  Space kick  |  Esc/X quit  |  " + GetPhysicsBackend())
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Or KeyDown(KEY_X) Then End
    EndIf
Wend
End
