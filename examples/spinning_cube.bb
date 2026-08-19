; CreateCube — 2026 command style (classic names still work)

Graphics3D(800, 600, 0, 2)
SetBuffer(BackBuffer())
SetWindowTitle("BitShin BASIC — Spinning cube")
SetCameraClsColor(18, 22, 32)
SetAmbientLight(70, 80, 100)

camera = CreateCamera()
SetPosition(camera, 0, 2, -6)
PointEntity(camera, 0, 0, 0)

light = CreateLight()
SetRotation(light, 50, 30, 0)

ground = CreatePlane(16, 16)
SetPosition(ground, 0, -1, 0)
SetEntityColor(ground, 36, 42, 52)

cube = CreateCube()
SetPosition(cube, 0, 0.2, 0)
SetEntityColor(cube, 70, 160, 255)

frames = 0
While True
    frames = frames + 1
    dt# = DeltaTime() * 60
    TurnEntity(cube, 0.5 * dt, 1 * dt, 0)
    RenderWorld
    Text(16, 16, "BitShin BASIC — Spinning cube  |  Esc quit")
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
