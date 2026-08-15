; Two cameras, two viewports. Click to CameraPick. Esc quits.

Graphics3D(800, 600)
SetBuffer(BackBuffer())
SetWindowTitle("BitShin BASIC — splitscreen")

camL = CreateCamera()
SetPosition(camL, -3, 2, -6)
SetCameraViewport(camL, 0, 0, 400, 600)

camR = CreateCamera()
SetPosition(camR, 3, 2, -6)
SetCameraViewport(camR, 400, 0, 400, 600)

light = CreateLight()
ground = CreatePlane()
SetEntityColor(ground, 40, 48, 58)

a = CreateCube()
SetPosition(a, -2, 0.5, 4)
SetEntityColor(a, 255, 90, 90)
SetEntityRadius(a, 1)

b = CreateSphere()
SetPosition(b, 2, 0.5, 4)
SetEntityColor(b, 80, 180, 255)
SetEntityRadius(b, 1)

While Not KeyDown(KEY_ESCAPE)
    dt# = DeltaTime() * 60
    TurnEntity(a, 0, 0.6 * dt, 0)
    If MouseHit(1) Then
        hit = CameraPick(camL, MouseX(), MouseY())
        If hit Then SetEntityColor(hit, Rand(80, 255), Rand(80, 255), Rand(80, 255))
    EndIf
    UpdateWorld
    RenderWorld
    Text(12, 12, "Splitscreen  |  click left view  |  Picked=" + GetRayHitEntity())
    Flip
Wend
End
