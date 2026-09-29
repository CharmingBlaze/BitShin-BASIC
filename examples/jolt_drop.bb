; Jolt drop — rigid bodies (Jolt by default; -tags nojolt for fallback).
; Space applies an impulse. Esc or X quits.

SetWindowTitle("BitShin BASIC — Jolt drop")
Graphics3D(960, 600, 0, 2)
SetCameraClsColor(18, 20, 28)

; Camera / light
cam = CreateCamera().Position([0, 6, -14]).Rotate(18, 0, 0)
CreateLight()

; World
ground = CreateCube().Scale(8, 0.2, 8).Position([0, 0, 10]).Color(50, 56, 70)
CreateRigidBodyBox(ground, 8, 0.2, 8, 0)

ball = CreateSphere(12).Position([0, 8, 10]).Color(80, 190, 255)
CreateRigidBodySphere(ball, 1, 1)

Print("Physics backend:", GetPhysicsBackend())

; Loop
While Not KeyDown(1) And Not KeyDown(KEY_X)
    If KeyHit(KEY_SPACE) Then ApplyImpulse(ball, 0, 6, 0)
    UpdateWorld
    hit = Raycast(0, 12, 10, 0, -20, 0)
    RenderWorld
    Text(16, 16, "Jolt drop  |  Space kick  |  Esc/X quit  |  " + GetPhysicsBackend())
    Text(16, 36, "Raycast hit " + hit + "  y=" + Int(PickedY()))
    Flip
Wend
End
