; Helicopter — collective, cyclic, and yaw.
; W/S collective, arrows cyclic, Q/E yaw.
; Esc quits after a few frames.

; window
SetWindowTitle("BitShin BASIC — Helicopter")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(70, 120, 170)
CreateLight()

; camera
cam = CreateCamera().Position([0, 10, -18]).Rotate(16, 0, 0)

; world
ground = CreateCube().Scale(50, 0.25, 50).Position([0, 0, 10]).Color(60, 90, 55)
CreateRigidBodyBox(ground, 50, 0.25, 50, 0)

; vehicle
heli = CreateCube().Scale(1.4, 0.4, 2.2).Position([0, 6, 10]).Color(70, 170, 90)
CreateHelicopterController(heli)

; loop
frames = 0
While 1
    frames = frames + 1
    col# = 0
    cp# = 0
    cr# = 0
    yaw# = 0
    If KeyDown(KEY_W) Then col = 0.55
    If KeyDown(KEY_S) Then col = -0.25
    If KeyDown(KEY_UP) Then cp = 0.4
    If KeyDown(KEY_DOWN) Then cp = -0.4
    If KeyDown(KEY_A) Then cr = -0.4
    If KeyDown(KEY_D) Then cr = 0.4
    If KeyDown(KEY_Q) Then yaw = -0.5
    If KeyDown(KEY_E) Then yaw = 0.5
    UpdateHelicopter(heli, col, cp, cr, yaw)
    UpdateWorld
    CameraFollow(cam, heli, 16, 5, 7, EntityYaw(heli), 14)
    RenderWorld
    Text(12, 12, "Heli  W/S collective  arrows cyclic  QE yaw")
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
