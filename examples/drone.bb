; Quad drone: hover + PD attitude. Esc after a few frames.

SetWindowTitle("BitShin BASIC — Drone")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(24, 28, 40)
CreateLight()

cam = CreateCamera()
SetPosition(cam, 0, 8, -14)
SetRotation(cam, 16, 0, 0)

ground = CreateCube()
SetScale(ground, 30, 0.25, 30)
SetPosition(ground, 0, 0, 8)
SetEntityColor(ground, 48, 52, 64)
CreateRigidBodyBox(ground, 30, 0.25, 30, 0)

drone = CreateCube()
SetScale(drone, 0.5, 0.12, 0.5)
SetPosition(drone, 0, 4, 8)
SetEntityColor(drone, 200, 80, 220)
CreateDroneController(drone)

frames = 0
While 1
    frames = frames + 1
    th# = 0.15
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
    RenderWorld
    Text(12, 12, "Drone  W/S thrust  arrows tilt  QE yaw")
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
