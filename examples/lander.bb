; BitShin BASIC — Lunar Lander / Rocket Vehicle Demo
SetWindowTitle("BitShin BASIC — Lunar Lander Physics")
Graphics3D(1280, 720, 0, 2)
SetBuffer(BackBuffer())
SetCameraClsColor(10, 12, 18)
SetAmbientLight(40, 45, 55)

cam = CreateCamera()
SetCameraRange(cam, 0.1, 1000)

sun = CreateDirectionalLight()
SetLightDirection(sun, 60, 45, 0)
SetLightColor(sun, 255, 250, 230)
SetLightShadow(sun, True)
EnableShadows(True)

; Lunar surface & Landing Pad
ground = CreateCube()
SetScale(ground, 80, 0.5, 80)
SetPosition(ground, 0, 0, 0)
SetEntityColor(ground, 100, 105, 115)
CreateRigidBodyBox(ground, 80, 0.5, 80, 0)

pad = CreateCylinder(16)
SetScale(pad, 6, 0.2, 6)
SetPosition(pad, 0, 0.6, 0)
SetEntityColor(pad, 220, 180, 50)
CreateRigidBodyBox(pad, 6, 0.2, 6, 0)

; Lander Module
lander = CreateCube()
SetScale(lander, 1.2, 1.0, 1.2)
SetPosition(lander, 0, 15, 0)
SetEntityColor(lander, 220, 220, 230)

thruster = CreateCone(8, lander)
SetScale(thruster, 0.6, 0.8, 0.6)
SetPosition(thruster, 0, -0.8, 0)
SetRotation(thruster, 180, 0, 0)
SetEntityColor(thruster, 60, 65, 70)

CreateLanderController(lander, 1.2, 1.0, 1.2, 1200)

frames = 0
While 1
    frames = frames + 1
    If frames > 8 And KeyHit(KEY_ESCAPE) Then Exit

    mainThrust# = 0.0
    If KeyDown(KEY_SPACE) Or KeyDown(KEY_W) Then mainThrust = 1.0

    pitch# = GetAxis(KEY_DOWN, KEY_UP)
    roll# = GetAxis(KEY_LEFT, KEY_RIGHT)
    yaw# = GetAxis(KEY_A, KEY_D)

    UpdateLander(lander, mainThrust#, pitch#, roll#, yaw#)

    CameraFollow(cam, lander, 18, 6, 10, EntityYaw(lander), 15)

    Cls
    UpdateWorld
    RenderWorld

    Text(20, 20, "Lunar Lander / Rocket Physics Controller")
    Text(20, 44, "Space / W: Main Engine Thrust | Arrows: Pitch/Roll RCS | A/D: Yaw RCS | Esc: Quit")
    Text(20, 68, "Altitude: " + Int(EntityY(lander)) + " m | Vertical Speed: " + Int(BodyVelocityY(lander)) + " m/s")
    Flip
Wend
End
