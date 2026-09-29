; Mech — heavy bipedal walker with torso turn and strafe.
; W/S walk, A/D turn, Q/E strafe.
; Esc quits after a few frames.

; window
SetWindowTitle("BitShin BASIC — Heavy Mech Walker")
Graphics3D(1280, 720, 0, 2)
SetBuffer(BackBuffer())
SetCameraClsColor(45, 55, 65)
SetAmbientLight(70, 75, 85)

; camera
cam = CreateCamera()
SetCameraRange(cam, 0.1, 1000)

sun = CreateDirectionalLight()
SetLightDirection(sun, 45, 40, 0)
SetLightColor(sun, 255, 235, 200)
SetLightShadow(sun, True)
EnableShadows(True)

; world
ground = CreateCube().Scale(60, 0.25, 60).Position([0, 0, 0]).Color(60, 70, 50)
CreateRigidBodyBox(ground, 60, 0.25, 60, 0)

; vehicle
mech = CreateCube().Scale(1.4, 1.8, 1.4).Position([0, 4, 0]).Color(160, 80, 50)

gunL = CreateCylinder(8, mech).Scale(0.2, 1.2, 0.2).Position([-1.2, 0.2, 0.8]).Rotate(90, 0, 0).Color(80, 85, 90)
gunR = CreateCylinder(8, mech).Scale(0.2, 1.2, 0.2).Position([1.2, 0.2, 0.8]).Rotate(90, 0, 0).Color(80, 85, 90)

CreateMechController(mech, 1.4, 1.8, 1.4, 4500)

; loop
frames = 0
camYaw# = 0.0
While 1
    frames = frames + 1
    If frames > 8 And KeyHit(KEY_ESCAPE) Then Exit

    throttle# = GetAxis(KEY_S, KEY_W)
    turn# = GetAxis(KEY_A, KEY_D)
    strafe# = GetAxis(KEY_Q, KEY_E)

    UpdateMech(mech, throttle#, turn#, strafe#)

    CameraFollow(cam, mech, 12, 5, 8, EntityYaw(mech), 18)

    Cls
    UpdateWorld
    RenderWorld

    Text(20, 20, "Heavy Bipedal Mech Walker Controller")
    Text(20, 44, "W/S: Walk Forward/Back | A/D: Turn Torso | Q/E: Strafe | Esc: Quit")
    Text(20, 68, "Speed: " + Int(BodyVelocity(mech)) + " | Yaw: " + Int(EntityYaw(mech)))
    Flip
Wend
End
