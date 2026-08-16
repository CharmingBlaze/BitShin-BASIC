; Collisions / SetEntityType — sphere vs cube

Graphics3D(800, 600, 0, 2)
SetBuffer(BackBuffer())
SetWindowTitle("BitShin BASIC — Bounce")
SetCameraClsColor(18, 22, 32)
SetAmbientLight(80, 90, 110)

cam = CreateCamera()
SetPosition(cam, 0, 8, -12)
SetRotation(cam, 28, 0, 0)

light = CreateLight()
SetLightDirection(light, 50, 30, 0)

floor = CreatePlane(24, 24)
SetPosition(floor, 0, 0, 10)
SetEntityColor(floor, 40, 48, 58)

ball = CreateSphere(12)
SetPosition(ball, -4, 1, 10)
SetEntityColor(ball, 80, 180, 255)
SetEntityType(ball, 1)
SetEntityRadius(ball, 1)

wall = CreateCube()
SetPosition(wall, 3, 1, 10)
SetEntityColor(wall, 255, 90, 90)
SetEntityType(wall, 2)
SetEntityRadius(wall, 1.4)

Collisions(1, 2, 1, 1)

vx# = 0.08
While Not KeyDown(1)
    dt# = DeltaTime() * 60
    MoveEntity(ball, vx * dt, 0, 0)
    UpdateWorld
    If CountCollisions(ball) > 0 Then
        vx = -vx
        SetEntityColor(ball, Rand(80, 255), Rand(80, 255), Rand(80, 255))
    EndIf
    RenderWorld
    Text(16, 16, "BitShin BASIC — Bounce  |  Esc quit")
    Flip
Wend
End
