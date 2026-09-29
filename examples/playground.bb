; Playground — primitives spin on a shared pivot; watch the colours move.
; Esc quits.

SetWindowTitle("BitShin BASIC — Playground")
Graphics3D(1100, 700, 0, 2)

SetCameraClsColor(12, 14, 22)
SetAmbientLight(40, 40, 50)

; Camera
cam = CreateCamera()
cam.Position([0, 3, -8])
cam.Rotate(12, 0, 0)

; Sun
sun = CreateLight(1)
sun.Position([4, 8, -3])
sun.Rotate(50, 30, 0)

; Ground
ground = CreatePlane().Position([0, -1.2, 8]).Color(36, 42, 56)

; Pivot + children
pivot = CreatePivot()
pivot.Position([0, 0, 8])

cube = CreateCube(pivot).Position([-3, 0, 0]).Color(255, 90, 80)
sphere = CreateSphere(16, pivot).Position([0, 0.2, 0]).Color(80, 220, 160)
cone = CreateCone(16, pivot).Position([3, 0, 0]).Color(255, 200, 70)

; Free torus (not parented)
torus = CreateTorus().Position([0, 2.2, 8]).Color(140, 160, 255)

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
