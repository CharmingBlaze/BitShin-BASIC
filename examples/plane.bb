; Plane — aero lift / drag with control torque.
; W throttle, arrows pitch, A/D roll, Q/E yaw.
; Esc quits after a few frames.

; window
SetWindowTitle("BitShin BASIC — Plane")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(70, 140, 210)
CreateLight()

; camera
cam = CreateCamera().Position([0, 12, -28]).Rotate(12, 0, 0)

; world
ground = CreateCube().Scale(80, 0.25, 80).Position([0, 0, 20]).Color(60, 110, 55)
CreateRigidBodyBox(ground, 80, 0.25, 80, 0)

; vehicle
plane = CreateCube().Scale(4, 0.25, 3).Position([0, 8, 20]).Color(230, 230, 240)
CreatePlaneController(plane)

; loop
frames = 0
While 1
    frames = frames + 1
    th# = 0.55
    pitch# = 0
    roll# = 0
    yaw# = 0
    If KeyDown(KEY_W) Then th = 1
    If KeyDown(KEY_S) Then th = 0.15
    If KeyDown(KEY_UP) Then pitch = 0.4
    If KeyDown(KEY_DOWN) Then pitch = -0.4
    If KeyDown(KEY_A) Then roll = -0.5
    If KeyDown(KEY_D) Then roll = 0.5
    If KeyDown(KEY_Q) Then yaw = -0.35
    If KeyDown(KEY_E) Then yaw = 0.35
    UpdatePlane(plane, th, pitch, roll, yaw)
    UpdateWorld
    CameraFollow(cam, plane, 22, 6, 6, EntityYaw(plane), 12)
    RenderWorld
    Text(12, 12, "Plane  W throttle  arrows pitch  AD roll  QE yaw")
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
