; First-person walk — WASD + mouse look

Graphics3D(800, 600, 0, 2)
SetBuffer(BackBuffer())
HideCursor()

cam = CreateCamera()
SetPosition(cam, 0, 1.6, 0)
SetCameraRange(cam, 0.1, 500)

light = CreateLight()
SetRotation(light, 60, 30, 0)
SetLightColor(light, 255, 255, 240)

ground = CreatePlane()
SetPosition(ground, 0, 0, 20)
SetEntityColor(ground, 40, 48, 62)

For i = 1 To 16
    block = CreateCube()
    SetPosition(block, Rand(-14, 14), 0.5, Rand(6, 26))
    SetEntityColor(block, Rand(80, 255), Rand(80, 220), Rand(80, 255))
Next

yaw# = 0
pitch# = 0
SetMousePosition(400, 300)

While Not KeyDown(1)
    dt# = DeltaTime() * 60
    mxs# = MouseDeltaX()
    mys# = MouseDeltaY()
    yaw = yaw + mxs * 0.15
    pitch = pitch - mys * 0.15
    If pitch > 80 Then pitch = 80
    If pitch < -80 Then pitch = -80
    SetRotation(cam, pitch, yaw, 0)
    SetMousePosition(400, 300)

    If KeyDown(17) Then MoveEntity(cam, 0, 0, 0.12 * dt)
    If KeyDown(31) Then MoveEntity(cam, 0, 0, -0.12 * dt)
    If KeyDown(30) Then MoveEntity(cam, -0.12 * dt, 0, 0)
    If KeyDown(32) Then MoveEntity(cam, 0.12 * dt, 0, 0)

    UpdateWorld
    RenderWorld
    Flip
Wend
End
