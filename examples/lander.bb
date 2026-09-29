; Lander — lunar / rocket physics with main engine and RCS.
; Space / W thrust, arrows pitch / roll, A/D yaw.
; Esc quits after a few frames.

; window
SetWindowTitle("BitShin BASIC — Lunar Lander Physics")
Graphics3D(1280, 720, 0, 2)
SetBuffer(BackBuffer())
SetCameraClsColor(10, 12, 18)
SetAmbientLight(40, 45, 55)

; camera
cam = CreateCamera()
SetCameraRange(cam, 0.1, 1000)

sun = CreateDirectionalLight()
SetLightDirection(sun, 60, 45, 0)
SetLightColor(sun, 255, 250, 230)
SetLightShadow(sun, True)
EnableShadows(True)

; world
ground = CreateCube().Scale(80, 0.5, 80).Position([0, 0, 0]).Color(100, 105, 115)
CreateRigidBodyBox(ground, 80, 0.5, 80, 0)

pad = CreateCylinder(16).Scale(6, 0.2, 6).Position([0, 0.6, 0]).Color(220, 180, 50)
CreateRigidBodyBox(pad, 6, 0.2, 6, 0)

; vehicle
lander = CreateCube().Scale(1.2, 1.0, 1.2).Position([0, 15, 0]).Color(220, 220, 230)

thruster = CreateCone(8, lander).Scale(0.6, 0.8, 0.6).Position([0, -0.8, 0]).Rotate(180, 0, 0).Color(60, 65, 70)

CreateLanderController(lander, 1.2, 1.0, 1.2, 1200)

; loop
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
