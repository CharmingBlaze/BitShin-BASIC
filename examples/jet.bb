; Jet — high thrust with weaker low-speed lift.
; W afterburner, S cut, arrows pitch, A/D roll, Q/E yaw.
; Esc quits after a few frames.

; window
SetWindowTitle("BitShin BASIC — Jet")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(40, 70, 110)
CreateLight()

; camera
cam = CreateCamera().Position([0, 14, -32]).Rotate(14, 0, 0)

; world
ground = CreateCube().Scale(90, 0.25, 90).Position([0, 0, 24]).Color(70, 90, 70)
CreateRigidBodyBox(ground, 90, 0.25, 90, 0)

; vehicle
jet = CreateCube().Scale(2.4, 0.28, 4.5).Position([0, 10, 24]).Color(180, 190, 210)
CreateJetController(jet)

; loop
frames = 0
While 1
    frames = frames + 1
    th# = 0.7
    pitch# = 0
    roll# = 0
    yaw# = 0
    If KeyDown(KEY_W) Then th = 1.2
    If KeyDown(KEY_S) Then th = 0.2
    If KeyDown(KEY_UP) Then pitch = 0.35
    If KeyDown(KEY_DOWN) Then pitch = -0.35
    If KeyDown(KEY_A) Then roll = -0.55
    If KeyDown(KEY_D) Then roll = 0.55
    If KeyDown(KEY_Q) Then yaw = -0.3
    If KeyDown(KEY_E) Then yaw = 0.3
    UpdateJet(jet, th, pitch, roll, yaw)
    UpdateWorld
    CameraFollow(cam, jet, 28, 7, 5, EntityYaw(jet), 12)
    RenderWorld
    Text(12, 12, "Jet  W afterburner  arrows pitch  AD roll  QE yaw")
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
