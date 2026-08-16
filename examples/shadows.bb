; Directional CSM plus point/spot maps and the extra filter commands.
; Esc quits. Headless CI covers the command logic; this is the visual check.

Graphics3D(960, 600)
SetWindowTitle("BitShin BASIC — Shadows")
SetCameraClsColor(12, 14, 22)
SetAmbientLight(28, 32, 42)

cam = CreateCamera()
SetPosition(cam, 0, 4, -10)
PointEntity(cam, 0, 0.4, 2)

EnableShadows True
ShadowCascades 2
ShadowMapSize 1024
SetShadowBias 0.003
SetShadowPCF 3
SetShadowFilter "pcss"
EnableShadowAtlas True
EnableShadowCache True
EnableContactShadows True
EnableScreenSpaceShadows True

sun = CreateDirectionalLight()
SetLightDirection sun, 50, 35, 0
SetLightColor sun, 255, 230, 190
SetLightShadow sun, True

lamp = CreatePointLight()
SetPosition(lamp, 2.2, 2.4, 3)
SetLightColor lamp, 80, 170, 255
SetLightRange lamp, 10
SetLightShadow lamp, True

spot = CreateSpotLight()
SetPosition(spot, -3, 4, 1)
SetLightDirection spot, -55, 20, 0
SetLightCone spot, 12, 35
SetLightColor spot, 255, 180, 90
SetLightRange spot, 14
SetLightShadow spot, True

ground = CreatePlane()
SetScale(ground, 16, 1, 16)
SetPosition(ground, 0, 0, 4)
SetEntityColor(ground, 48, 54, 64)

cube = CreateCube()
SetPosition(cube, 0, 0.7, 4)
SetEntityColor(cube, 220, 90, 70)

ball = CreateSphere()
SetPosition(ball, 2.4, 0.7, 5)
SetEntityColor(ball, 80, 200, 140)

Print "Shadows: directional + point + spot  |  1=PCF 2=PCSS 3=EVSM 4=MSM"

While Not KeyDown(1)
    dt# = DeltaTime() * 60
    TurnEntity(cube, 0.3 * dt, 0.5 * dt, 0)
    If KeyHit(KEY_1) Then SetShadowFilter "pcf"
    If KeyHit(KEY_2) Then SetShadowFilter "pcss"
    If KeyHit(KEY_3) Then SetShadowEVSM True
    If KeyHit(KEY_4) Then SetShadowMSM True
    UpdateWorld
    RenderWorld
    Flip
Wend
End
