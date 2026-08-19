; BitShin BASIC — Heavy Mech Walker Vehicle Demo
SetWindowTitle("BitShin BASIC — Heavy Mech Walker")
Graphics3D(1280, 720, 0, 2)
SetBuffer(BackBuffer())
SetCameraClsColor(45, 55, 65)
SetAmbientLight(70, 75, 85)

cam = CreateCamera()
SetCameraRange(cam, 0.1, 1000)

sun = CreateDirectionalLight()
SetLightDirection(sun, 45, 40, 0)
SetLightColor(sun, 255, 235, 200)
SetLightShadow(sun, True)
EnableShadows(True)

ground = CreateCube()
SetScale(ground, 60, 0.25, 60)
SetPosition(ground, 0, 0, 0)
SetEntityColor(ground, 60, 70, 50)
CreateRigidBodyBox(ground, 60, 0.25, 60, 0)

; Mech Body
mech = CreateCube()
SetScale(mech, 1.4, 1.8, 1.4)
SetPosition(mech, 0, 4, 0)
SetEntityColor(mech, 160, 80, 50)

gunL = CreateCylinder(8, mech)
SetScale(gunL, 0.2, 1.2, 0.2)
SetPosition(gunL, -1.2, 0.2, 0.8)
SetRotation(gunL, 90, 0, 0)
SetEntityColor(gunL, 80, 85, 90)

gunR = CreateCylinder(8, mech)
SetScale(gunR, 0.2, 1.2, 0.2)
SetPosition(gunR, 1.2, 0.2, 0.8)
SetRotation(gunR, 90, 0, 0)
SetEntityColor(gunR, 80, 85, 90)

CreateMechController(mech, 1.4, 1.8, 1.4, 4500)

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
