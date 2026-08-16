; Lights — directional + point, SetEntityColor, SetAmbientLight

Graphics3D(640, 480)
SetBuffer(BackBuffer())
SetWindowTitle("BitShin BASIC — Lights")
SetCameraClsColor(8, 10, 16)
SetAmbientLight(20, 20, 28)
SetCameraFogMode(1)
SetCameraFogColor(8, 10, 16)
SetCameraFogRange(8, 40)

cam = CreateCamera()
SetPosition(cam, 0, 2, -6)

EnableShadows True
ShadowCascades 2
SetShadowFilter "pcf"
EnableShadowAtlas True

sun = CreateLight(1)
SetRotation(sun, 50, 30, 0)
SetLightColor(sun, 255, 230, 180)
SetLightShadow sun, True

lamp = CreateLight(2)
SetPosition(lamp, 2, 2, 4)
SetLightColor(lamp, 80, 160, 255)
SetLightRange(lamp, 12)
SetLightShadow lamp, True

cube = CreateCube()
SetPosition(cube, 0, 0, 5)
SetEntityColor(cube, 255, 255, 255)
SetMaterialShininess(cube, 0.6)

While Not KeyDown(1)
    dt# = DeltaTime() * 60
    TurnEntity(cube, 0.3 * dt, 0.5 * dt, 0)
    RenderWorld
    Flip
Wend
End
