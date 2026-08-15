; Detour agents + local crowd separation (go-detour has no DetourCrowd).

Graphics3D(960, 600)
SetWindowTitle("Crowd")
cam = CreateCamera()
SetPosition(cam, 0, 18, -20)
PointEntity(cam, 0, 0, 0)
light = CreateLight()
SetRotation(light, 80, 0, 0)

floor = CreateCube()
SetScale(floor, 14, 0.2, 14)
SetEntityColor(floor, 50, 70, 60)

nav = CreateNavMesh(floor)
BakeNavMesh(nav)
cr = CreateCrowd(1.3)

For i = 0 To 7
    b = CreateCube()
    SetScale(b, 0.35, 0.7, 0.35)
    SetPosition(b, -6 + i * 1.5, 1, -5)
    SetEntityColor(b, 80 + i * 15, 140, 220)
    CreateAgent(b)
    CrowdAddAgent(cr, b)
Next

CrowdSetDestination(cr, 6, 1, 6)

While Not KeyDown(1)
    UpdateWorld
    CrowdUpdate()
    RenderWorld
    Flip
Wend
End
