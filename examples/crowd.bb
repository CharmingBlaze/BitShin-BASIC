; Crowd — eight Detour agents walk to a shared destination with local separation.
; Esc quits.

Graphics3D(960, 600)
SetWindowTitle("BitShin BASIC — Crowd")
SetCameraClsColor(28, 36, 48)
SetAmbientLight(90, 100, 120)

; Camera / light
cam = CreateCamera().Position([0, 16, -22])
CameraRange(cam, 0.15, 4000)
cam.Point(0, 1, 0)
light = CreateLight().Rotate(55, 30, 0)

; World
floor = CreateCube().Scale(14, 0.2, 14).Position([0, 0, 0]).Color(50, 70, 60)

SetNavMaxSlope(45)
nav = CreateNavMesh(floor)
BakeNavMesh(nav)
cr = CreateCrowd(1.3)

For i = 0 To 7
    b = CreateCube().Scale(0.45, 0.9, 0.45).Position([-6 + i * 1.5, 1, -5]).Color(80 + i * 20, 160, 255)
    CreateAgent(b)
    SetAgentSpeed(b, 5)
    CrowdAddAgent(cr, b)
Next

CrowdSetDestination(cr, 6, 1, 6)

; Loop
While Not KeyDown(1)
    UpdateWorld
    CrowdUpdate()
    RenderWorld
    Text(16, 16, "BitShin BASIC — Crowd  |  8 agents walk to (6,1,6)  |  Esc quit")
    Flip
Wend
End
