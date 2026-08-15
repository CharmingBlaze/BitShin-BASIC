; Boat: Gerstner WaterHeight buoyancy at hull offsets. Esc after a few frames.

SetWindowTitle("BitShin BASIC — Boat")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(40, 90, 130)
CreateLight()

cam = CreateCamera()
SetPosition(cam, 0, 10, -18)
SetRotation(cam, 18, 0, 0)

water = CreateWater(220, 220, 48)
SetWaterLevel(0)
SetGerstner(0, 0.9, 0.3, 0.28, 0.45, 18, 1.1)

boat = CreateCube()
SetScale(boat, 1.6, 0.32, 3.0)
SetPosition(boat, 0, 1.2, 8)
SetEntityColor(boat, 190, 95, 50)
CreateBoatController(boat)
SetLinearDamping(boat, 1.8)

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
    RenderWorld
    Text(12, 12, "Boat  WASD  waterY=" + Str(WaterHeight(0, 8)))
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
