; BitShin BASIC — Modern Dynamic Shadows Showcase
SetWindowTitle("BitShin BASIC — Real-Time Game Shadows")
Graphics3D(1280, 720, 0, 2)
SetBuffer(BackBuffer())
SetCameraClsColor(45, 55, 75)
SetAmbientLight(65, 70, 85)

cam = CreateCamera()
SetCameraRange(cam, 0.1, 1000)
SetPosition(cam, 0, 5.5, -8.5)
PointEntity(cam, 0, 0.8, 3.5)

EnableShadows True
ShadowCascades 2
ShadowMapSize 2048
SetShadowBias 0.002
SetShadowPCF 3
SetShadowFilter "pcf"

; Directional Sunlight
sun = CreateDirectionalLight()
SetLightDirection sun, 50, 40, 0
SetLightColor sun, 255, 240, 210
SetLightShadow sun, True

; Point Lamp
lamp = CreatePointLight()
SetPosition(lamp, 3.0, 3.0, 3.0)
SetLightColor lamp, 100, 180, 255
SetLightRange lamp, 12
SetLightShadow lamp, True

; Spot Light
spot = CreateSpotLight()
SetPosition(spot, -4.0, 4.5, 1.0)
SetLightDirection spot, -45, 30, 0
SetLightCone spot, 15, 45
SetLightColor spot, 255, 190, 100
SetLightRange spot, 16
SetLightShadow spot, True

; Arena Floor
ground = CreateCube()
SetScale(ground, 20, 0.25, 20)
SetPosition(ground, 0, 0, 3.5)
SetEntityColor(ground, 65, 80, 70)
EntityShininess(ground, 0.05)

; Central Rotating Cube
cube = CreateCube()
SetScale(cube, 1.2, 1.2, 1.2)
SetPosition(cube, 0, 1.2, 3.5)
SetEntityColor(cube, 230, 80, 60)
EntityShininess(cube, 0.2)
EntitySpecular(cube, 60, 60, 60)

; Sphere
ball = CreateSphere(16)
SetScale(ball, 1.0, 1.0, 1.0)
SetPosition(ball, 2.6, 1.0, 4.5)
SetEntityColor(ball, 70, 190, 130)
EntityShininess(ball, 0.3)
EntitySpecular(ball, 80, 80, 80)

; Torus Ring
ring = CreateTorus(1.4, 0.25, 16, 24)
SetPosition(ring, -2.8, 1.8, 3.5)
SetRotation(ring, 45, 30, 0)
SetEntityColor(ring, 240, 180, 50)
EntityShininess(ring, 0.4)
EntitySpecular(ring, 100, 100, 100)

mode$ = "1: 16-Tap Poisson PCF"
frames = 0

While 1
    frames = frames + 1
    If frames > 8 And KeyHit(KEY_ESCAPE) Then Exit
    dt# = DeltaTime() * 60

    TurnEntity(cube, 0.4 * dt, 0.6 * dt, 0.2 * dt)
    TurnEntity(ring, 0.3 * dt, 0.5 * dt, 0)

    If KeyHit(KEY_1) Then
        SetShadowFilter "pcf"
        mode$ = "1: 16-Tap Poisson PCF"
    EndIf
    If KeyHit(KEY_2) Then
        SetShadowFilter "pcss"
        mode$ = "2: Contact-Hardening PCSS"
    EndIf
    If KeyHit(KEY_3) Then
        SetShadowEVSM True
        mode$ = "3: Exponential Variance (EVSM)"
    EndIf
    If KeyHit(KEY_4) Then
        SetShadowMSM True
        mode$ = "4: Moment Shadows (MSM)"
    EndIf

    Cls
    UpdateWorld
    RenderWorld

    Text(20, 20, "BitShin BASIC — Real-Time Game Shadows Pipeline")
    Text(20, 44, "1: PCF | 2: PCSS (Variable Soft Penumbra) | 3: EVSM | 4: MSM | Esc: Quit")
    Text(20, 68, "Active Shadow Filtering: " + mode$)
    Flip
Wend
End
