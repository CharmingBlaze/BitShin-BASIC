; Jolt cloth sheet (pinned top) + water buoyancy/flow. Esc to quit.

Graphics3D(960, 540, 0, 2)
SetWindowTitle("BitShin BASIC — Cloth + Water")
SetCameraClsColor(70, 110, 150)
SetAmbientLight(70, 76, 88)

cam = CreateCamera()
SetPosition(cam, 0, 6, -14)
CameraLookAt(cam, 0, 3, 4)

sun = CreateDirectionalLight()
SetLightDirection(sun, 40, 50, 10)
SetLightColor(sun, 255, 236, 200)

water = CreateWater(80, 80, 32)
SetWaterStyle("gerstner")
SetWaterLevel(0)
SetWaterWind(0.9, 0.2, 0.6)
SetWaterFlow(1.4, 0, 0.2)

ground = CreateCube()
SetScale(ground, 16, 0.3, 16)
SetPosition(ground, 0, -0.2, 8)
SetEntityColor(ground, 62, 88, 58)
CreateRigidBodyBox(ground, 16, 0.3, 16, 0)

pole = CreateCube()
SetScale(pole, 0.12, 4, 0.12)
SetPosition(pole, -1, 5, 6)
SetEntityColor(pole, 140, 110, 70)
CreateRigidBodyBox(pole, 0.12, 4, 0.12, 0)

flag = CreateCloth(2.2, 1.4, 10, 8, 1)
SetPosition(flag, -1, 6.8, 6)
SetEntityColor(flag, 210, 48, 42)
SetClothWind(flag, 0.8)

crate = CreateCube()
SetScale(crate, 0.7, 0.7, 0.7)
SetPosition(crate, 2, 3, 6)
SetEntityColor(crate, 196, 140, 62)
CreateRigidBodyBox(crate, 0.7, 0.7, 0.7, 12)
SetBuoyancyFactor(crate, 1.2)

frames = 0
While 1
    UpdateWorld()
    RenderWorld()
    Flip()
    frames = frames + 1
    If KeyHit(1) Or frames > 8 Then End
Wend
