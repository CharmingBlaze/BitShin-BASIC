; Hovercraft — skirt lift with yaw steer.
; WASD throttle / steer.
; Esc quits after a few frames.

; window
SetWindowTitle("BitShin BASIC — Hovercraft")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(40, 70, 90)
CreateLight()

; camera
cam = CreateCamera().Position([0, 8, -16]).Rotate(18, 0, 0)

; world
ground = CreateCube().Scale(50, 0.25, 50).Position([0, 0, 10]).Color(70, 80, 70)
CreateRigidBodyBox(ground, 50, 0.25, 50, 0)

; vehicle
hover = CreateCube().Scale(1.8, 0.22, 2.2).Position([0, 1.0, 10]).Color(90, 180, 220)
CreateHovercraftController(hover)

; loop
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
    CameraFollow(cam, hover, 14, 4, 8, EntityYaw(hover), 16)
    RenderWorld
    Text(12, 12, "Hovercraft  WASD  Esc quit")
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
