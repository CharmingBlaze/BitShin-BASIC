; Top-down twin-stick — WASD move, mouse aim, Space dash.
; Esc quits after a few frames.

SetWindowTitle("BitShin BASIC — Modern Top-Down Action Controller")
Graphics3D(1280, 720, 0, 2)
SetBuffer(BackBuffer())
SetCameraClsColor(30, 35, 45)
SetAmbientLight(65, 70, 80)

; Camera
cam = CreateCamera()
SetCameraRange(cam, 0.1, 1000)

; Sun + shadows
sun = CreateDirectionalLight()
SetLightDirection(sun, 60, 30, 0)
SetLightColor(sun, 255, 230, 200)
SetLightShadow(sun, True)
EnableShadows(True)

; Arena
ground = CreateCube().Scale(40, 0.25, 40).Position([0, 0, 0]).Color(45, 55, 65)
CreateRigidBodyBox(ground, 40, 0.25, 40, 0)

; Player (body + turret)
player = CreatePivot()
player.Position([0, 1, 0])

body = CreateBox(0.8, 0.8, 1.2, player).Position([0, 0.5, 0]).Color(70, 180, 240)

turret = CreateCylinder(8, player).Scale(0.15, 0.8, 0.15).Position([0, 0.6, 0.6]).Rotate(90, 0, 0).Color(255, 200, 50)

CreateTopDownController(player, 8.5, 20.0)

frames = 0
While 1
    frames = frames + 1
    If frames > 8 And KeyHit(KEY_ESCAPE) Then Exit

    moveX# = GetAxis(KEY_A, KEY_D)
    moveZ# = GetAxis(KEY_S, KEY_W)

    aimX# = GetMouseWorldX#(cam, 1.0)
    aimZ# = GetMouseWorldZ#(cam, 1.0)
    dash = KeyHit(KEY_SPACE)

    UpdateTopDown(player, moveX#, moveZ#, aimX#, aimZ#, dash)

    px# = EntityX(player)
    pz# = EntityZ(player)
    CameraFollow(cam, player, 16, 22, 12, 0, 55)

    Cls
    UpdateWorld
    RenderWorld

    Text(20, 20, "Modern Top-Down / Twin-Stick Action Controller")
    Text(20, 44, "WASD: Move | Mouse: Aim Target | Space: Dash | Esc: Quit")
    Text(20, 68, "Player Aim Yaw: " + Int(GetTopDownAimYaw#(player)) + " | Pos: (" + Int(px) + ", " + Int(pz) + ")")
    Flip
Wend
End
