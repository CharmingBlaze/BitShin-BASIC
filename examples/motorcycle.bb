; 2-wheel Jolt MotorcycleController. Esc after a few frames.

SetWindowTitle("BitShin BASIC — Motorcycle")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(28, 32, 40)
CreateLight()

cam = CreateCamera()
SetPosition(cam, 0, 6, -12)
SetRotation(cam, 18, 0, 0)

ground = CreateCube()
SetScale(ground, 40, 0.25, 40)
SetPosition(ground, 0, 0, 8)
SetEntityColor(ground, 52, 58, 64)
CreateRigidBodyBox(ground, 40, 0.25, 40, 0)

bike = CreateCube()
SetScale(bike, 0.28, 0.5, 1.2)
SetPosition(bike, 0, 1.2, 8)
SetEntityColor(bike, 240, 200, 40)
CreateMotorcycleController(bike)

frames = 0
While 1
    frames = frames + 1
    steer# = 0
    throttle# = 0
    brake# = 0
    If KeyDown(KEY_A) Then steer = -1
    If KeyDown(KEY_D) Then steer = 1
    If KeyDown(KEY_W) Then throttle = 1
    If KeyDown(KEY_S) Then throttle = -0.3
    If KeyDown(KEY_SPACE) Then brake = 1
    UpdateMotorcycle(bike, steer, throttle, brake)
    UpdateWorld
    RenderWorld
    Text(12, 12, "Motorcycle  WASD  Space brake  Esc quit")
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
