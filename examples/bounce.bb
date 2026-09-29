; Bounce — sphere vs cube via Collisions / SetEntityType.
; Esc quits.

Graphics3D(800, 600, 0, 2)
SetBuffer(BackBuffer())
SetWindowTitle("BitShin BASIC — Bounce")
SetCameraClsColor(18, 22, 32)
SetAmbientLight(80, 90, 110)

; Camera / light
cam = CreateCamera().Position([0, 8, -12]).Rotate(28, 0, 0)
light = CreateLight()
SetLightDirection(light, 50, 30, 0)

; World
floor = CreatePlane(24, 24).Position([0, 0, 10]).Color(40, 48, 58)

ball = CreateSphere(12).Position([-4, 1, 10]).Color(80, 180, 255)
SetEntityType(ball, 1)
SetEntityRadius(ball, 1)

wall = CreateCube().Position([3, 1, 10]).Color(255, 90, 90)
SetEntityType(wall, 2)
SetEntityRadius(wall, 1.4)

Collisions(1, 2, 1, 1)

; Loop — reverse velocity on hit
vx# = 0.08
While Not KeyDown(1)
    dt# = DeltaTime() * 60
    ball.Move(vx * dt, 0, 0)
    UpdateWorld
    If CountCollisions(ball) > 0 Then
        vx = -vx
        ball.Color(Rand(80, 255), Rand(80, 255), Rand(80, 255))
    EndIf
    RenderWorld
    Text(16, 16, "BitShin BASIC — Bounce  |  Esc quit")
    Flip
Wend
End
