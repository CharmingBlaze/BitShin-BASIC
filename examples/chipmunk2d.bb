; Chipmunk2D (jakecoffman/cp) — X/Y plane

SetWindowTitle("BitShin BASIC — Chipmunk 2D")
Graphics3D(960, 600, 0, 2)
SetCameraClsColor(16, 18, 26)

cam = CreateCamera()
SetPosition(cam, 0, 0, -16)
CreateLight()

Physics2D()
Gravity2D(0, -220)

floor = CreateCube()
SetScale(floor, 8, 0.3, 0.3)
SetPosition(floor, 0, -4, 0)
SetEntityColor(floor, 60, 66, 80)
CreateBox2D(floor, 16, 0.6, 0, 0)

ball = CreateSphere(10)
SetPosition(ball, -2, 5, 0)
SetEntityColor(ball, 255, 170, 70)
CreateCircle2D(ball, 1, 1, 1)

crate = CreateCube()
SetScale(crate, 0.5, 0.5, 0.5)
SetPosition(crate, 2, 3, 0)
SetEntityColor(crate, 180, 90, 70)
CreateBox2D(crate, 1, 1, 1, 1)
CreatePin2D(floor, crate)

While Not KeyDown(1) And Not KeyDown(KEY_X)
    If KeyHit(KEY_SPACE) Then ApplyImpulse2D(ball, 0, 180)
    UpdateWorld
    hit = Raycast2D(-4, 6, 4, -4)
    RenderWorld
    Text(16, 16, "Chipmunk2D  |  Space hops  |  Esc/X quit  |  ray " + hit)
    Flip
Wend
End
