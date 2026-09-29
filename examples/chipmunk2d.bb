; Chipmunk2D — floor, bouncing ball, and a pinned crate on the X/Y plane.
; Space hops the ball. Esc or X quits.

SetWindowTitle("BitShin BASIC — Chipmunk 2D")
Graphics3D(960, 600, 0, 2)
SetCameraClsColor(16, 18, 26)

; Camera / light
cam = CreateCamera().Position([0, 0, -16])
CreateLight()

; Physics
Physics2D()
Gravity2D(0, -220)

; World
floor = CreateCube().Scale(8, 0.3, 0.3).Position([0, -4, 0]).Color(60, 66, 80)
CreateBox2D(floor, 16, 0.6, 0, 0)

ball = CreateSphere(10).Position([-2, 5, 0]).Color(255, 170, 70)
CreateCircle2D(ball, 1, 1, 1)

crate = CreateCube().Scale(0.5, 0.5, 0.5).Position([2, 3, 0]).Color(180, 90, 70)
CreateBox2D(crate, 1, 1, 1, 1)
CreatePin2D(floor, crate)

; Loop
While Not KeyDown(1) And Not KeyDown(KEY_X)
    If KeyHit(KEY_SPACE) Then ApplyImpulse2D(ball, 0, 180)
    UpdateWorld
    hit = Raycast2D(-4, 6, 4, -4)
    RenderWorld
    Text(16, 16, "Chipmunk2D  |  Space hops  |  Esc/X quit  |  ray " + hit)
    Flip
Wend
End
