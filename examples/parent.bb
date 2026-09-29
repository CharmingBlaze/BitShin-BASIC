; Parent — moon orbits a spinning planet via SetEntityParent
; Esc quits.

Graphics3D(640, 480)
SetBuffer(BackBuffer())
SetWindowTitle("BitShin BASIC — Parent")

; Camera
camera = CreateCamera()
SetPosition(camera, 0, 3, -8)
SetRotation(camera, 12, 0, 0)

; Light
light = CreateLight()
SetRotation(light, 45, 30, 0)

; Scene — planet parent, moon child (local Position/Scale)
ground = CreatePlane(16, 16)
SetPosition(ground, 0, -1.2, 8)
SetEntityColor(ground, 36, 42, 52)

planet = CreateSphere(16)
SetPosition(planet, 0, 0, 8)
SetEntityColor(planet, 70, 140, 220)

moon = CreateSphere(8)
SetEntityColor(moon, 220, 200, 120)
SetEntityParent(moon, planet)
SetPosition(moon, 2.4, 0.4, 0)
SetScale(moon, 0.35, 0.35, 0.35)

; Loop — both turn; child follows parent transform
While Not KeyDown(1)
    dt# = DeltaTime() * 60
    TurnEntity(planet, 0, 0.6 * dt, 0)
    TurnEntity(moon, 0, 1.2 * dt, 0)
    RenderWorld
    Flip
Wend
End
