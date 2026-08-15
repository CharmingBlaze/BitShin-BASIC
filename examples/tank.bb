; Tracked tank via Jolt TrackedVehicleController. Esc after a few frames.

SetWindowTitle("BitShin BASIC — Tank")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(36, 40, 32)
CreateLight()

cam = CreateCamera()
SetPosition(cam, 0, 8, -16)
SetRotation(cam, 20, 0, 0)

ground = CreateCube()
SetScale(ground, 40, 0.25, 40)
SetPosition(ground, 0, 0, 10)
SetEntityColor(ground, 70, 78, 58)
CreateRigidBodyBox(ground, 40, 0.25, 40, 0)

tank = CreateCube()
SetScale(tank, 1.3, 0.45, 2.4)
SetPosition(tank, 0, 1.1, 10)
SetEntityColor(tank, 90, 100, 70)
CreateTankController(tank)

frames = 0
While 1
    frames = frames + 1
    steer# = 0
    throttle# = 0
    brake# = 0
    If KeyDown(KEY_A) Then steer = -1
    If KeyDown(KEY_D) Then steer = 1
    If KeyDown(KEY_W) Then throttle = 1
    If KeyDown(KEY_S) Then throttle = -0.4
    If KeyDown(KEY_SPACE) Then brake = 1
    UpdateTank(tank, steer, throttle, brake)
    UpdateWorld
    RenderWorld
    Text(12, 12, "Tank  WASD  Space brake  Esc quit")
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
