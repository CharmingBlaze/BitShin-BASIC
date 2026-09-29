; FPS controller — WASD move, Shift sprint, Ctrl crouch, Space jump, mouse look.
; Esc quits after a few frames.

SetWindowTitle("BitShin BASIC — Modern FPS Controller")
Graphics3D(1280, 720, 0, 2)
SetBuffer(BackBuffer())
SetCameraClsColor(40, 50, 70)
SetAmbientLight(60, 65, 80)
HidePointer()

; Camera
cam = CreateCamera()
SetCameraRange(cam, 0.1, 1000)

; Sun + shadows
sun = CreateDirectionalLight()
SetLightDirection(sun, 45, 35, 0)
SetLightColor(sun, 255, 240, 210)
SetLightShadow(sun, True)
EnableShadows(True)

; Arena ground
ground = CreateCube().Scale(60, 0.25, 60).Position([0, 0, 0]).Color(45, 80, 60)
CreateRigidBodyBox(ground, 60, 0.25, 60, 0)

; Pillars in a ring
For i = 0 To 7
    ang# = i * 45.0
    px# = Sin(ang) * 22.0
    pz# = Cos(ang) * 22.0
    pillar = CreateCylinder(12).Scale(1.2, 4.0, 1.2).Position([px, 4.0, pz]).Color(180, 150, 110)
    CreateRigidBodyBox(pillar, 1.2, 4.0, 1.2, 0)
Next

; Player + FPS controller
player = CreatePivot()
player.Position([0, 2, 0])
CreateFPSController(player, cam, 1.8, 0.4, 6.5, 11.5, 9.5)

frames = 0
While 1
    frames = frames + 1
    If frames > 8 And KeyHit(KEY_ESCAPE) Then Exit

    dt# = DeltaTime()
    walkX# = GetAxis(KEY_A, KEY_D)
    walkZ# = GetAxis(KEY_S, KEY_W)
    jump = KeyHit(KEY_SPACE)
    sprint = KeyDown(KEY_LEFT_SHIFT)
    crouch = KeyDown(KEY_LEFT_CONTROL)

    UpdateFPS(player, walkX#, walkZ#, jump, sprint, crouch, 0.18)

    Cls
    UpdateWorld
    RenderWorld

    Text(20, 20, "Modern FPS Character Controller")
    Text(20, 44, "WASD: Move | Shift: Sprint | Ctrl: Crouch | Space: Jump | Mouse: Look | Esc: Quit")
    Text(20, 68, "Grounded: " + IsFPSGrounded(player) + " | Pitch: " + Int(GetFPSPitch#(player)) + " | Yaw: " + Int(GetFPSYaw#(player)))
    Flip
Wend
End
