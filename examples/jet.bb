; Jet: high thrust, weaker low-speed lift. Esc after a few frames.

SetWindowTitle("BitShin BASIC — Jet")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(40, 70, 110)
CreateLight()

cam = CreateCamera()
SetPosition(cam, 0, 14, -32)
SetRotation(cam, 14, 0, 0)

ground = CreateCube()
SetScale(ground, 90, 0.25, 90)
SetPosition(ground, 0, 0, 24)
SetEntityColor(ground, 70, 90, 70)
CreateRigidBodyBox(ground, 90, 0.25, 90, 0)

jet = CreateCube()
SetScale(jet, 2.4, 0.28, 4.5)
SetPosition(jet, 0, 10, 24)
SetEntityColor(jet, 180, 190, 210)
CreateJetController(jet)

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
    RenderWorld
    Text(12, 12, "Jet  W afterburner  arrows pitch  AD roll  QE yaw")
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
