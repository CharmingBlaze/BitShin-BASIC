; 3D Showcase — rotating cube over a ground plane
; Esc quits. Press F5 to Run, F7 to Build Standalone.

Graphics3D(1280, 720, 0, 2)
SetWindowTitle("BitShin BASIC — 3D Showcase")

; Camera — PositionEntity + CameraClsColor
camera = CreateCamera()
PositionEntity(camera, 0, 2, -6)
CameraClsColor(15, 18, 26)

; Light — sun above the horizon, warm tint
light = CreateLight(1)
SetLightDirection(light, 50, 35, 0)
SetLightColor(light, 255, 240, 220)

; Scene — EntityColor on cube and plane
cube = CreateCube()
EntityColor(cube, 56, 189, 248)

plane = CreatePlane(40, 40)
PositionEntity(plane, 0, -1, 0)
EntityColor(plane, 30, 41, 59)

Print("Game loop started. Use ESC to quit.")

; Loop — spin cube with DeltaTime scaling; KeyDown(1) is Esc
While Not KeyDown(1)
    dt# = DeltaTime() * 60
    TurnEntity(cube, 0.4 * dt, 0.7 * dt, 0)

    RenderWorld
    Flip
Wend
End
