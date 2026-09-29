; Spaceship — zero-g torque and local impulse.
; WASD / QE attitude, Space boost, R yaw pulse.
; Esc quits after a few frames.

; window
SetWindowTitle("BitShin BASIC — Spaceship")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(6, 8, 16)
CreateLight()

; camera
cam = CreateCamera().Position([0, 6, -18]).Rotate(10, 0, 0)

; vehicle
ship = CreateCube().Scale(1.4, 0.45, 2.2).Position([0, 4, 8]).Color(120, 200, 255)
CreateSpaceshipController(ship)
SetGravityScale(ship, 0)

; loop
frames = 0
While 1
    frames = frames + 1
    th# = 0
    pitch# = 0
    roll# = 0
    yaw# = 0
    If KeyDown(KEY_W) Then th = 1
    If KeyDown(KEY_S) Then th = -0.5
    If KeyDown(KEY_UP) Then pitch = 1
    If KeyDown(KEY_DOWN) Then pitch = -1
    If KeyDown(KEY_A) Then yaw = -1
    If KeyDown(KEY_D) Then yaw = 1
    If KeyDown(KEY_Q) Then roll = -1
    If KeyDown(KEY_E) Then roll = 1
    UpdateSpaceship(ship, th, pitch, roll, yaw)
    If KeyHit(KEY_SPACE) Then ApplyLocalImpulse(ship, 0, 0, 12)
    If KeyHit(KEY_R) Then ApplyTorque(ship, 0, 800, 0)
    UpdateWorld
    CameraFollow(cam, ship, 16, 4, 6, EntityYaw(ship), 12)
    RenderWorld
    Text(12, 12, "Spaceship  WASD/QE  Space boost  R yaw pulse")
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
