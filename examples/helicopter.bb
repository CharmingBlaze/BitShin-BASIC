; Helicopter: collective + cyclic + yaw. Esc after a few frames.

SetWindowTitle("BitShin BASIC — Helicopter")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(70, 120, 170)
CreateLight()

cam = CreateCamera()
SetPosition(cam, 0, 10, -18)
SetRotation(cam, 16, 0, 0)

ground = CreateCube()
SetScale(ground, 50, 0.25, 50)
SetPosition(ground, 0, 0, 10)
SetEntityColor(ground, 60, 90, 55)
CreateRigidBodyBox(ground, 50, 0.25, 50, 0)

heli = CreateCube()
SetScale(heli, 1.4, 0.4, 2.2)
SetPosition(heli, 0, 6, 10)
SetEntityColor(heli, 70, 170, 90)
CreateHelicopterController(heli)

frames = 0
While 1
    frames = frames + 1
    col# = 0.1
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
    RenderWorld
    Text(12, 12, "Heli  W/S collective  arrows cyclic  QE yaw")
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
