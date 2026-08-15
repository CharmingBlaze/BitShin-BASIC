; Aero plane: lift/drag + torque. Esc after a few frames.

SetWindowTitle("BitShin BASIC — Plane")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(70, 140, 210)
CreateLight()

cam = CreateCamera()
SetPosition(cam, 0, 12, -28)
SetRotation(cam, 12, 0, 0)

ground = CreateCube()
SetScale(ground, 80, 0.25, 80)
SetPosition(ground, 0, 0, 20)
SetEntityColor(ground, 60, 110, 55)
CreateRigidBodyBox(ground, 80, 0.25, 80, 0)

plane = CreateCube()
SetScale(plane, 4, 0.25, 3)
SetPosition(plane, 0, 8, 20)
SetEntityColor(plane, 230, 230, 240)
CreatePlaneController(plane)

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
    RenderWorld
    Text(12, 12, "Plane  W throttle  arrows pitch  AD roll  QE yaw")
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
