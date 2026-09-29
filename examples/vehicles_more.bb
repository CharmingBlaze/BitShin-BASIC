; More vehicles — bike, heli, hover, sub, tank, and drone together.
; WASD drives shared throttle / steer inputs.
; Esc quits after a few frames.

; window
SetWindowTitle("BitShin BASIC — More vehicles")
Graphics3D(1100, 640, 0, 2)
SetCameraClsColor(30, 40, 52)
CreateLight()

; camera
cam = CreateCamera().Position([0, 14, -28]).Rotate(22, 0, 0)

; world
ground = CreateCube().Scale(50, 0.25, 50).Position([0, 0, 12]).Color(52, 64, 58)
CreateRigidBodyBox(ground, 50, 0.25, 50, 0)

water = CreateWater(80, 80, 24)
SetWaterLevel(-1.2)
water.Position([18, -1.2, 12])

; vehicle
bike = CreateCube().Scale(0.28, 0.5, 1.2).Position([-10, 1.2, 8]).Color(240, 200, 40)
CreateMotorcycleController(bike)

heli = CreateCube().Scale(1.4, 0.4, 2.2).Position([-4, 6, 8]).Color(70, 170, 90)
CreateHelicopterController(heli)

hover = CreateCube().Scale(1.8, 0.22, 2.2).Position([2, 1.0, 8]).Color(90, 180, 220)
CreateHovercraftController(hover)

sub = CreateCube().Scale(1.0, 0.55, 3.4).Position([16, 0.2, 12]).Color(40, 90, 110)
CreateSubmarineController(sub)

tank = CreateCube().Scale(1.3, 0.45, 2.4).Position([8, 1.1, 6]).Color(90, 100, 70)
CreateTankController(tank)

drone = CreateCube().Scale(0.5, 0.12, 0.5).Position([0, 5, 4]).Color(200, 80, 220)
CreateDroneController(drone)

; loop
frames = 0
While 1
    frames = frames + 1
    steer# = 0
    th# = 0.35
    If KeyDown(KEY_A) Then steer = -1
    If KeyDown(KEY_D) Then steer = 1
    If KeyDown(KEY_W) Then th = 1
    If KeyDown(KEY_S) Then th = -0.3
    UpdateMotorcycle(bike, steer, th, 0)
    UpdateHelicopter(heli, 0.15, 0, 0, steer * 0.4)
    UpdateHovercraft(hover, th, steer)
    UpdateSubmarine(sub, th * 0.4, steer, 0.1)
    UpdateTank(tank, steer, th, 0)
    UpdateDrone(drone, 0.2, 0, 0, steer * 0.3)
    UpdateWorld
    RenderWorld
    Text(12, 12, "bike heli hover sub tank drone  WASD  Esc quit")
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
