; Car — 4-wheel Jolt VehicleConstraint drive.
; WASD steer / throttle, Space brake.
; Esc quits after a few frames.

; window
SetWindowTitle("BitShin BASIC — Car")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(22, 28, 36)
CreateLight()

; camera
cam = CreateCamera().Position([0, 8, -16]).Rotate(22, 0, 0)

; world
ground = CreateCube().Scale(40, 0.25, 40).Position([0, 0, 10]).Color(48, 56, 64)
CreateRigidBodyBox(ground, 40, 0.25, 40, 0)

; vehicle
car = CreateCube().Scale(1.1, 0.35, 2.0).Position([0, 1.2, 10]).Color(210, 70, 55)
CreateCarController(car)

; loop
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
    CameraFollow(cam, car, 14, 4.5, 8, EntityYaw(car), 16)
    RenderWorld
    Text(12, 12, "Car  WASD drive  Space brake  Esc quit")
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
