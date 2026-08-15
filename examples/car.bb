; 4-wheel Jolt VehicleConstraint. Esc after a few frames.

SetWindowTitle("BitShin BASIC — Car")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(22, 28, 36)
CreateLight()

cam = CreateCamera()
SetPosition(cam, 0, 8, -16)
SetRotation(cam, 22, 0, 0)

ground = CreateCube()
SetScale(ground, 40, 0.25, 40)
SetPosition(ground, 0, 0, 10)
SetEntityColor(ground, 48, 56, 64)
CreateRigidBodyBox(ground, 40, 0.25, 40, 0)

car = CreateCube()
SetScale(car, 1.1, 0.35, 2.0)
SetPosition(car, 0, 1.2, 10)
SetEntityColor(car, 210, 70, 55)
CreateCarController(car)

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
    UpdateCar(car, steer, throttle, brake)
    UpdateWorld
    RenderWorld
    Text(12, 12, "Car  WASD drive  Space brake  Esc quit")
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
