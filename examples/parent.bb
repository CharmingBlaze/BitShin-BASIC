; SetEntityParent — moon orbits a spinning planet

Graphics3D(640, 480)
SetBuffer(BackBuffer())
SetWindowTitle("BitShin BASIC — Parent")

camera = CreateCamera()
SetPosition(camera, 0, 3, -8)
SetRotation(camera, 12, 0, 0)

light = CreateLight()
SetRotation(light, 45, 30, 0)

planet = CreateSphere(16)
SetPosition(planet, 0, 0, 8)
SetEntityColor(planet, 70, 140, 220)

moon = CreateSphere(8)
SetEntityColor(moon, 220, 200, 120)
SetEntityParent(moon, planet)
SetPosition(moon, 2.4, 0.4, 0)
SetScale(moon, 0.35, 0.35, 0.35)

While Not KeyDown(1)
    dt# = DeltaTime() * 60
    TurnEntity(planet, 0, 0.6 * dt, 0)
    TurnEntity(moon, 0, 1.2 * dt, 0)
    RenderWorld
    Flip
Wend
End
