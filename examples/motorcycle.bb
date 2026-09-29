; Motorcycle — 2-wheel Jolt MotorcycleController.
; WASD steer / throttle, Space brake.
; Esc quits after a few frames.

; window
SetWindowTitle("BitShin BASIC — Motorcycle")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(28, 32, 40)
CreateLight()

; camera
cam = CreateCamera().Position([0, 6, -12]).Rotate(18, 0, 0)

; world
ground = CreateCube().Scale(40, 0.25, 40).Position([0, 0, 8]).Color(52, 58, 64)
CreateRigidBodyBox(ground, 40, 0.25, 40, 0)

; vehicle
bike = CreateCube().Scale(0.28, 0.5, 1.2).Position([0, 1.2, 8]).Color(240, 200, 40)
CreateMotorcycleController(bike)

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
    If KeyDown(KEY_S) Then throttle = -0.3
    If KeyDown(KEY_SPACE) Then brake = 1
    UpdateMotorcycle(bike, steer, throttle, brake)
    UpdateWorld
    CameraFollow(cam, bike, 10, 3.2, 8, EntityYaw(bike), 16)
    RenderWorld
    Text(12, 12, "Motorcycle  WASD  Space brake  Esc quit")
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
