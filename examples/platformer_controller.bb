; 2.5D platformer — A/D or arrows run; Space / Up jump (double + variable height).
; Esc quits after a few frames.

SetWindowTitle("BitShin BASIC — Modern 2.5D Platformer")
Graphics3D(1280, 720, 0, 2)
SetBuffer(BackBuffer())
SetCameraClsColor(70, 110, 150)
SetAmbientLight(80, 85, 95)

; Camera
cam = CreateCamera()
SetCameraRange(cam, 0.1, 1000)

; Sun + shadows
sun = CreateDirectionalLight()
SetLightDirection(sun, 45, 30, 0)
SetLightColor(sun, 255, 245, 220)
SetLightShadow(sun, True)
EnableShadows(True)

; Ground
ground = CreateCube().Scale(40, 0.5, 4).Position([0, 0, 0]).Color(70, 130, 80)
CreateRigidBodyBox(ground, 40, 0.5, 4, 0)

; Floating platforms
Function AddPlatform(px#, py#, pz#, sx#, sy#, sz#, r, g, b)
    p = CreateCube().Scale(sx, sy, sz).Position([px, py, pz]).Color(r, g, b)
    CreateRigidBodyBox(p, sx, sy, sz, 0)
End Function

AddPlatform(6, 2.5, 0, 3, 0.3, 2, 180, 140, 90)
AddPlatform(14, 5.0, 0, 3, 0.3, 2, 180, 140, 90)
AddPlatform(-8, 3.5, 0, 3, 0.3, 2, 180, 140, 90)
AddPlatform(-16, 6.5, 0, 3, 0.3, 2, 180, 140, 90)

; Player
player = CreatePivot()
player.Position([0, 2, 0])

pMesh = CreateCapsule(0.4, 1.2, 12, player).Position([0, 0.8, 0]).Color(240, 90, 60)

CreatePlatformerController(player, 8.5, 12.0, 2)

frames = 0
While 1
    frames = frames + 1
    If frames > 8 And KeyHit(KEY_ESCAPE) Then Exit

    moveX# = GetAxis(KEY_A, KEY_D)
    If KeyDown(KEY_LEFT) Then moveX = -1.0
    If KeyDown(KEY_RIGHT) Then moveX = 1.0

    jumpPressed = KeyHit(KEY_SPACE) Or KeyHit(KEY_UP)
    jumpHeld = KeyDown(KEY_SPACE) Or KeyDown(KEY_UP)

    UpdatePlatformer(player, moveX#, jumpPressed, jumpHeld)

    ; Side-view tracking camera
    px# = EntityX(player)
    py# = EntityY(player)
    CameraSmoothLook(cam, px#, py# + 1.5, 0, 10)
    SetPosition(cam, px#, py# + 3.5, -16)

    Cls
    UpdateWorld
    RenderWorld

    Text(20, 20, "Modern 2.5D Platformer Controller")
    Text(20, 44, "A/D or Arrows: Run | Space / Up: Jump (Double Jump + Variable Jump Height) | Esc: Quit")
    Text(20, 68, "Grounded: " + IsPlatformerGrounded(player) + " | Pos: (" + Int(px) + ", " + Int(py) + ")")
    Flip
Wend
End
