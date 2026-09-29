; TPS controller — WASD move, Shift sprint, RMB aim, Space jump; spring-arm camera.
; Esc quits after a few frames.

SetWindowTitle("BitShin BASIC — Modern TPS Controller & SpringArm")
Graphics3D(1280, 720, 0, 2)
SetBuffer(BackBuffer())
SetCameraClsColor(50, 70, 95)
SetAmbientLight(70, 75, 90)
HidePointer()

; Camera
cam = CreateCamera()
SetCameraRange(cam, 0.1, 1000)

; Sun + shadows
sun = CreateDirectionalLight()
SetLightDirection(sun, 50, 40, 0)
SetLightColor(sun, 255, 235, 200)
SetLightShadow(sun, True)
EnableShadows(True)

; Ground (spring-arm collides with geometry)
ground = CreateCube().Scale(50, 0.25, 50).Position([0, 0, 0]).Color(50, 90, 70)
CreateRigidBodyBox(ground, 50, 0.25, 50, 0)

; Wall for camera collision avoidance
wall1 = CreateCube().Scale(8, 4, 0.5).Position([0, 4, 12]).Color(140, 110, 80)
CreateRigidBodyBox(wall1, 8, 4, 0.5, 0)

; Player mesh hierarchy
player = CreatePivot()
player.Position([0, 2, 0])

body = CreateCylinder(12, player).Scale(0.5, 0.65, 0.5).Position([0, 0.8, 0]).Color(60, 150, 220)
head = CreateSphere(10, player).Scale(0.35, 0.35, 0.35).Position([0, 1.6, 0]).Color(255, 210, 160)

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
