; Several primitives, parenting, and colour

SetWindowTitle("BitShin BASIC — Playground")
Graphics3D(1100, 700, 0, 2)

SetCameraClsColor(12, 14, 22)
SetAmbientLight(40, 40, 50)

cam = CreateCamera()
SetPosition(cam, 0, 3, -8)
SetRotation(cam, 12, 0, 0)

sun = CreateLight(1)
SetPosition(sun, 4, 8, -3)
SetRotation(sun, 50, 30, 0)

ground = CreatePlane()
SetPosition(ground, 0, -1.2, 8)
SetEntityColor(ground, 36, 42, 56)

pivot = CreatePivot()
SetPosition(pivot, 0, 0, 8)

cube = CreateCube(pivot)
SetPosition(cube, -3, 0, 0)
SetEntityColor(cube, 255, 90, 80)

sphere = CreateSphere(16, pivot)
SetPosition(sphere, 0, 0.2, 0)
SetEntityColor(sphere, 80, 220, 160)

cone = CreateCone(16, pivot)
SetPosition(cone, 3, 0, 0)
SetEntityColor(cone, 255, 200, 70)

torus = CreateTorus()
SetPosition(torus, 0, 2.2, 8)
SetEntityColor(torus, 140, 160, 255)

t# = 0
While Not KeyDown(KEY_ESCAPE)
    dt# = DeltaTime() * 60
    t = t + dt
    TurnEntity(pivot, 0, 0.4 * dt, 0)
    TurnEntity(cube, 0.6 * dt, 0.4 * dt, 0)
    TurnEntity(torus, 0.3 * dt, 0.8 * dt, 0.2 * dt)
    SetPosition(sphere, 0, 0.2 + Sin(t) * 0.4, 0)
    RenderWorld
    Text(16, 16, "WASD not needed — just watch. Esc quits.")
    Flip
Wend
End
