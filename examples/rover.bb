; Rover — first-person walk through a sparse block field.
; WASD move, mouse look.
; Esc quits.

; window
Graphics3D(800, 600, 0, 2)
SetBuffer(BackBuffer())
SetWindowTitle("BitShin BASIC — Rover")
SetCameraClsColor(70, 110, 160)
SetAmbientLight(90, 100, 120)
HideCursor()

; camera
cam = CreateCamera().Position([0, 1.6, 0])
SetCameraRange(cam, 0.1, 500)

; world
light = CreateLight()
light.Rotate(60, 30, 0)
SetLightColor(light, 255, 255, 240)

ground = CreatePlane(80, 80).Position([0, 0, 0]).Color(40, 48, 62)

For i = 1 To 16
    block = CreateCube().Position([Rand(-14, 14), 0.5, Rand(4, 22)]).Color(Rand(80, 255), Rand(80, 220), Rand(80, 255))
Next

; loop
yaw# = 0
pitch# = 0
frames = 0
SetMousePosition(400, 300)

While Not KeyDown(1)
    dt# = DeltaTime() * 60
    frames = frames + 1
    If frames > 2 Then
        mxs# = MouseDeltaX()
        mys# = MouseDeltaY()
        yaw = yaw + mxs * 0.15
        pitch = pitch - mys * 0.15
        If pitch > 80 Then pitch = 80
        If pitch < -80 Then pitch = -80
        cam.Rotate(pitch, yaw, 0)
        SetMousePosition(400, 300)
    EndIf

    If KeyDown(KEY_W) Or KeyDown(17) Then cam.Move(0, 0, 0.12 * dt)
    If KeyDown(KEY_S) Or KeyDown(31) Then cam.Move(0, 0, -0.12 * dt)
    If KeyDown(KEY_A) Or KeyDown(30) Then cam.Move(-0.12 * dt, 0, 0)
    If KeyDown(KEY_D) Or KeyDown(32) Then cam.Move(0.12 * dt, 0, 0)

    UpdateWorld
    RenderWorld
    Text(16, 16, "BitShin BASIC — Rover  |  WASD + mouse  |  Esc quit")
    Flip
Wend
End
