; Detour agents + local crowd separation (go-detour has no DetourCrowd).

Graphics3D(960, 600)
SetWindowTitle("BitShin BASIC — Crowd")
SetCameraClsColor(28, 36, 48)
SetAmbientLight(90, 100, 120)

cam = CreateCamera()
SetPosition(cam, 0, 16, -22)
CameraRange(cam, 0.15, 4000)
PointEntity(cam, 0, 1, 0)
light = CreateLight()
SetRotation(light, 55, 30, 0)

floor = CreateCube()
SetScale(floor, 14, 0.2, 14)
SetPosition(floor, 0, 0, 0)
SetEntityColor(floor, 50, 70, 60)

SetNavMaxSlope(45)
nav = CreateNavMesh(floor)
BakeNavMesh(nav)
cr = CreateCrowd(1.3)

For i = 0 To 7
    b = CreateCube()
    SetScale(b, 0.45, 0.9, 0.45)
    SetPosition(b, -6 + i * 1.5, 1, -5)
    SetEntityColor(b, 80 + i * 20, 160, 255)
    CreateAgent(b)
    SetAgentSpeed(b, 5)
    CrowdAddAgent(cr, b)
Next

CrowdSetDestination(cr, 6, 1, 6)

While Not KeyDown(1)
    UpdateWorld
    CrowdUpdate()
    RenderWorld
    Text(16, 16, "BitShin BASIC — Crowd  |  8 agents walk to (6,1,6)  |  Esc quit")
    Flip
Wend
End
