; CreateCube — 2026 command style (classic names still work)

Graphics3D(640, 480)
SetBuffer(BackBuffer())

camera = CreateCamera()
light = CreateLight()
SetRotation(light, 90, 0, 0)

cube = CreateCube()
SetPosition(cube, 0, 0, 5)
SetEntityColor(cube, 70, 160, 255)

While Not KeyDown(1)
    dt# = DeltaTime() * 60
    TurnEntity(cube, 0.5 * dt, 1 * dt, 0)
    RenderWorld
    Flip
Wend
End
