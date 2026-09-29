; Boat — Gerstner WaterHeight buoyancy at hull offsets.
; WASD throttle / steer.
; Esc quits after a few frames.

; window
SetWindowTitle("BitShin BASIC — Boat")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(40, 90, 130)
SetAmbientLight(90, 110, 130)
CreateLight()

; camera
cam = CreateCamera().Position([0, 8, -14]).Rotate(16, 0, 0)

; world
water = CreateWater(220, 220, 48)
SetWaterLevel(0)
SetGerstner(0, 0.9, 0.3, 0.28, 0.45, 18, 1.1)

; vehicle
boat = CreateCube().Scale(1.8, 0.55, 3.4).Position([0, 1.4, 6]).Color(230, 120, 50)
CreateBoatController(boat)
SetLinearDamping(boat, 1.8)
cam.Point(boat)

; loop
frames = 0
While 1
    frames = frames + 1
    th# = 0
    steer# = 0
    If KeyDown(KEY_W) Then th = 1
    If KeyDown(KEY_S) Then th = -0.4
    If KeyDown(KEY_A) Then steer = -1
    If KeyDown(KEY_D) Then steer = 1
    UpdateBoat(boat, th, steer)
    UpdateWorld
    CameraFollow(cam, boat, 16, 5, 7, EntityYaw(boat), 14)
    RenderWorld
    Text(12, 12, "BitShin BASIC — Boat  |  WASD  waterY=" + Str(WaterHeight(0, 8)))
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
