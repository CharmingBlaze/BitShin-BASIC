; Tank — tracked drive via Jolt TrackedVehicleController.
; WASD or arrows throttle / steer, Space brake.
; Esc quits after a few frames.

; window
SetWindowTitle("BitShin BASIC — Tank")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(36, 40, 32)
SetAmbientLight(80, 90, 80)
CreateLight()

; camera
cam = CreateCamera().Position([0, 10, -18])

; world
ground = CreateCube().Scale(40, 0.25, 40).Position([0, 0, 10]).Color(70, 78, 58)
CreateRigidBodyBox(ground, 40, 0.25, 40, 0)

; vehicle
tank = CreateCube().Scale(1.3, 0.45, 2.4).Position([0, 1.1, 10]).Color(90, 140, 70)
CreateTankController(tank)
cam.Point(tank)

; loop
frames = 0
While 1
    frames = frames + 1
    steer# = 0
    throttle# = 0
    brake# = 0
    If KeyDown(KEY_A) Or KeyDown(KEY_LEFT) Then steer = -1
    If KeyDown(KEY_D) Or KeyDown(KEY_RIGHT) Then steer = 1
    If KeyDown(KEY_W) Or KeyDown(KEY_UP) Then throttle = 1
    If KeyDown(KEY_S) Or KeyDown(KEY_DOWN) Then throttle = -0.4
    If KeyDown(KEY_SPACE) Then brake = 1
    UpdateTank(tank, steer, throttle, brake)
    UpdateWorld
    CameraFollow(cam, tank, 16, 5, 7, EntityYaw(tank), 16)
    RenderWorld
    Text(12, 12, "BitShin BASIC — Tank  |  WASD / arrows  Space brake  |  Esc quit")
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
