; Extra controllers: bike, heli, hover, sub, tank, drone. Esc after a few frames.

SetWindowTitle("BitShin BASIC — More vehicles")
Graphics3D(1100, 640, 0, 2)
SetCameraClsColor(30, 40, 52)
CreateLight()

cam = CreateCamera()
SetPosition(cam, 0, 14, -28)
SetRotation(cam, 22, 0, 0)

ground = CreateCube()
SetScale(ground, 50, 0.25, 50)
SetPosition(ground, 0, 0, 12)
SetEntityColor(ground, 52, 64, 58)
CreateRigidBodyBox(ground, 50, 0.25, 50, 0)

water = CreateWater(80, 80, 24)
SetWaterLevel(-1.2)
SetPosition(water, 18, -1.2, 12)

bike = CreateCube()
SetScale(bike, 0.28, 0.5, 1.2)
SetPosition(bike, -10, 1.2, 8)
SetEntityColor(bike, 240, 200, 40)
CreateMotorcycleController(bike)

heli = CreateCube()
SetScale(heli, 1.4, 0.4, 2.2)
SetPosition(heli, -4, 6, 8)
SetEntityColor(heli, 70, 170, 90)
CreateHelicopterController(heli)

hover = CreateCube()
SetScale(hover, 1.8, 0.22, 2.2)
SetPosition(hover, 2, 1.0, 8)
SetEntityColor(hover, 90, 180, 220)
CreateHovercraftController(hover)

sub = CreateCube()
SetScale(sub, 1.0, 0.55, 3.4)
SetPosition(sub, 16, 0.2, 12)
SetEntityColor(sub, 40, 90, 110)
CreateSubmarineController(sub)

tank = CreateCube()
SetScale(tank, 1.3, 0.45, 2.4)
SetPosition(tank, 8, 1.1, 6)
SetEntityColor(tank, 90, 100, 70)
CreateTankController(tank)

drone = CreateCube()
SetScale(drone, 0.5, 0.12, 0.5)
SetPosition(drone, 0, 5, 4)
SetEntityColor(drone, 200, 80, 220)
CreateDroneController(drone)

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
