; Many one-at-a-time bodies make Jolt's broadphase tree deep.
; OptimizePhysics rebuilds it after a spawn burst (coins, debris, piles).
; Space kicks the pile. Esc after a few frames, or X.

SetWindowTitle("BitShin BASIC — Physics pile")
Graphics3D(960, 600, 0, 2)
SetCameraClsColor(18, 20, 28)

cam = CreateCamera()
SetPosition(cam, 0, 10, -16)
SetRotation(cam, 22, 0, 0)
CreateLight()

ground = CreateCube()
SetScale(ground, 10, 0.2, 10)
SetPosition(ground, 0, 0, 8)
SetEntityColor(ground, 50, 56, 70)
CreateRigidBodyBox(ground, 10, 0.2, 10, 0)

first = 0
For i = 0 To 79
    b = CreateSphere(8)
    SetScale(b, 0.28, 0.28, 0.28)
    SetPosition(b, Rnd(6) - 3, 3 + i * 0.22, 8 + Rnd(6) - 3)
    SetEntityColor(b, 70 + Rand(120), 140 + Rand(80), 220)
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
