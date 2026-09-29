; Drone — quad hover with PD attitude control.
; W/S thrust, arrows tilt, Q/E yaw.
; Esc quits after a few frames.

; window
SetWindowTitle("BitShin BASIC — Drone")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(24, 28, 40)
CreateLight()

; camera
cam = CreateCamera().Position([0, 8, -14]).Rotate(16, 0, 0)

; world
ground = CreateCube().Scale(30, 0.25, 30).Position([0, 0, 8]).Color(48, 52, 64)
CreateRigidBodyBox(ground, 30, 0.25, 30, 0)

; vehicle
drone = CreateCube().Scale(0.5, 0.12, 0.5).Position([0, 4, 8]).Color(200, 80, 220)
CreateDroneController(drone)

; loop
frames = 0
While 1
    frames = frames + 1
    th# = 0
    cp# = 0
    cr# = 0
    yaw# = 0
    If KeyDown(KEY_W) Then th = 0.8
    If KeyDown(KEY_S) Then th = -0.2
    If KeyDown(KEY_UP) Then cp = 0.5
    If KeyDown(KEY_DOWN) Then cp = -0.5
    If KeyDown(KEY_A) Then cr = -0.5
    If KeyDown(KEY_D) Then cr = 0.5
    If KeyDown(KEY_Q) Then yaw = -0.6
    If KeyDown(KEY_E) Then yaw = 0.6
    UpdateDrone(drone, th, cp, cr, yaw)
    UpdateWorld
    CameraFollow(cam, drone, 10, 3.5, 8, EntityYaw(drone), 16)
    RenderWorld
    Text(12, 12, "Drone  W/S thrust  arrows tilt  QE yaw")
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
