; Shadows — directional, point, and spot lights with filter modes on a small arena.
; 1 PCF, 2 PCSS, 3 EVSM, 4 MSM. Esc quits (after a short startup delay).

SetWindowTitle("BitShin BASIC — Real-Time Game Shadows")
Graphics3D(1280, 720, 0, 2)
SetBuffer(BackBuffer())
SetCameraClsColor(45, 55, 75)
SetAmbientLight(65, 70, 85)

; Camera
cam = CreateCamera()
SetCameraRange(cam, 0.1, 1000)
cam.Position([0, 5.5, -8.5])
cam.Point(0, 0.8, 3.5)

; Shadow pipeline
EnableShadows True
ShadowCascades 2
ShadowMapSize 2048
SetShadowBias 0.002
SetShadowPCF 3
SetShadowFilter "pcf"

; Lights
sun = CreateDirectionalLight()
SetLightDirection sun, 50, 40, 0
SetLightColor sun, 255, 240, 210
SetLightShadow sun, True

lamp = CreatePointLight().Position([3.0, 3.0, 3.0])
SetLightColor lamp, 100, 180, 255
SetLightRange lamp, 12
SetLightShadow lamp, True

spot = CreateSpotLight().Position([-4.0, 4.5, 1.0])
SetLightDirection spot, -45, 30, 0
SetLightCone spot, 15, 45
SetLightColor spot, 255, 190, 100
SetLightRange spot, 16
SetLightShadow spot, True

; Arena
ground = CreateCube().Scale(20, 0.25, 20).Position([0, 0, 3.5]).Color(65, 80, 70)
ground.Shininess(0.05)

cube = CreateCube().Scale(1.2, 1.2, 1.2).Position([0, 1.2, 3.5]).Color(230, 80, 60)
cube.Shininess(0.2)
cube.Specular(60, 60, 60)

ball = CreateSphere(16).Scale(1.0, 1.0, 1.0).Position([2.6, 1.0, 4.5]).Color(70, 190, 130)
ball.Shininess(0.3)
ball.Specular(80, 80, 80)

ring = CreateTorus(1.4, 0.25, 16, 24).Position([-2.8, 1.8, 3.5]).Rotate(45, 30, 0).Color(240, 180, 50)
ring.Shininess(0.4)
ring.Specular(100, 100, 100)

mode$ = "1: 16-Tap Poisson PCF"
frames = 0

; Loop
While 1
    frames = frames + 1
    If frames > 8 And KeyHit(KEY_ESCAPE) Then Exit
    dt# = DeltaTime() * 60

    cube.Turn(0.4 * dt, 0.6 * dt, 0.2 * dt)
    ring.Turn(0.3 * dt, 0.5 * dt, 0)

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
