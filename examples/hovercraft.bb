; Hovercraft: skirt lift + yaw steer. Esc after a few frames.

SetWindowTitle("BitShin BASIC — Hovercraft")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(40, 70, 90)
CreateLight()

cam = CreateCamera()
SetPosition(cam, 0, 8, -16)
SetRotation(cam, 18, 0, 0)

ground = CreateCube()
SetScale(ground, 50, 0.25, 50)
SetPosition(ground, 0, 0, 10)
SetEntityColor(ground, 70, 80, 70)
CreateRigidBodyBox(ground, 50, 0.25, 50, 0)

hover = CreateCube()
SetScale(hover, 1.8, 0.22, 2.2)
SetPosition(hover, 0, 1.0, 10)
SetEntityColor(hover, 90, 180, 220)
CreateHovercraftController(hover)

frames = 0
While 1
    frames = frames + 1
    th# = 0
    steer# = 0
    If KeyDown(KEY_W) Then th = 1
    If KeyDown(KEY_S) Then th = -0.4
    If KeyDown(KEY_A) Then steer = -1
    If KeyDown(KEY_D) Then steer = 1
    UpdateHovercraft(hover, th, steer)
    UpdateWorld
    RenderWorld
    Text(12, 12, "Hovercraft  WASD  Esc quit")
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
