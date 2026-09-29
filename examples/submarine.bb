; Submarine — throttle, steer, and dive vs WaterHeight.
; WASD move / steer, Up / Down dive.
; Esc quits after a few frames.

; window
SetWindowTitle("BitShin BASIC — Submarine")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(12, 30, 48)
CreateLight()

; camera
cam = CreateCamera().Position([0, 8, -18]).Rotate(14, 0, 0)

; world
water = CreateWater(160, 160, 36)
SetWaterLevel(0)
SetGerstner(0, 0.6, 0.22, 0.2, 0.35, 16, 0.9)

; vehicle
sub = CreateCube().Scale(1.0, 0.55, 3.4).Position([0, 0.4, 8]).Color(40, 90, 110)
CreateSubmarineController(sub)

; loop
frames = 0
While 1
    frames = frames + 1
    th# = 0
    steer# = 0
    dive# = 0
    If KeyDown(KEY_W) Then th = 1
    If KeyDown(KEY_S) Then th = -0.4
    If KeyDown(KEY_A) Then steer = -1
    If KeyDown(KEY_D) Then steer = 1
    If KeyDown(KEY_DOWN) Then dive = 1
    If KeyDown(KEY_UP) Then dive = -1
    UpdateSubmarine(sub, th, steer, dive)
    UpdateWorld
    CameraFollow(cam, sub, 16, 5, 7, EntityYaw(sub), 14)
    RenderWorld
    Text(12, 12, "Sub  WASD  Up/Down dive  waterY=" + Str(WaterHeight(0, 8)))
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
