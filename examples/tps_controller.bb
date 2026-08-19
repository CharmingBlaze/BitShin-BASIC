; BitShin BASIC — Modern Third-Person Action Controller (TPS) Demo
SetWindowTitle("BitShin BASIC — Modern TPS Controller & SpringArm")
Graphics3D(1280, 720, 0, 2)
SetBuffer(BackBuffer())
SetCameraClsColor(50, 70, 95)
SetAmbientLight(70, 75, 90)
HidePointer()

cam = CreateCamera()
SetCameraRange(cam, 0.1, 1000)

sun = CreateDirectionalLight()
SetLightDirection(sun, 50, 40, 0)
SetLightColor(sun, 255, 235, 200)
SetLightShadow(sun, True)
EnableShadows(True)

; Ground & Walls to demonstrate SpringArm Camera Collision Avoidance
ground = CreateCube()
SetScale(ground, 50, 0.25, 50)
SetPosition(ground, 0, 0, 0)
SetEntityColor(ground, 50, 90, 70)
CreateRigidBodyBox(ground, 50, 0.25, 50, 0)

wall1 = CreateCube()
SetScale(wall1, 8, 4, 0.5)
SetPosition(wall1, 0, 4, 12)
SetEntityColor(wall1, 140, 110, 80)
CreateRigidBodyBox(wall1, 8, 4, 0.5, 0)

; Player mesh hierarchy
player = CreatePivot()
SetPosition(player, 0, 2, 0)

body = CreateCylinder(12, player)
SetScale(body, 0.5, 0.65, 0.5)
SetPosition(body, 0, 0.8, 0)
SetEntityColor(body, 60, 150, 220)

head = CreateSphere(10, player)
SetScale(head, 0.35, 0.35, 0.35)
SetPosition(head, 0, 1.6, 0)
SetEntityColor(head, 255, 210, 160)

CreateTPSController(player, cam, 1.8, 0.45, 6.5, 11.0, 10.0)

camPitch# = 15.0
camYaw# = 0.0

frames = 0
While 1
    frames = frames + 1
    If frames > 8 And KeyHit(KEY_ESCAPE) Then Exit

    camYaw = camYaw + MouseDeltaX() * 0.18
    camPitch = Clamp(camPitch + MouseDeltaY() * 0.14, -20, 65)

    moveX# = GetAxis(KEY_A, KEY_D)
    moveZ# = GetAxis(KEY_S, KEY_W)
    jump = KeyHit(KEY_SPACE)
    sprint = KeyDown(KEY_LEFT_SHIFT)
    aim = MouseDown(2)

    UpdateTPS(player, moveX#, moveZ#, jump, sprint, aim, 4.5, camPitch, camYaw)

    Cls
    UpdateWorld
    RenderWorld

    Text(20, 20, "Modern TPS Action Controller with Spring-Arm Camera")
    Text(20, 44, "WASD: Move | Shift: Sprint | Right Mouse: Aim Mode | Space: Jump | Esc: Quit")
    Text(20, 68, "Aim Mode: " + aim + " | Player Facing Yaw: " + Int(GetTPSYaw#(player)) + " | Grounded: " + IsTPSGrounded(player))
    Flip
Wend
End
