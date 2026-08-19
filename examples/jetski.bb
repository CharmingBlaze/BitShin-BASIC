; BitShin BASIC — classic arcade JetSki demo on modern water + Jolt physics.

SetWindowTitle("BitShin BASIC — JetSki")
Graphics3D(1280, 720, 0, 2)
SetBuffer(BackBuffer())
SetCameraClsColor(74, 151, 211)
SetAmbientLight(82, 94, 112)
HidePointer()
MoveMouse(640, 360)

cam = CreateCamera()
SetCameraRange(cam, 0.2, 2600)
CreateSkyBox("default")
CameraFogMode(cam, 1)
CameraFogColor(cam, 142, 188, 220)
CameraFogRange(cam, 150, 620)
SetWeather("clear")

sun = CreateDirectionalLight()
SetLightDirection(sun, 48, 34, 0)
SetLightColor(sun, 255, 237, 202)
SetLightShadow(sun, True)
EnableShadows(True)

; Rolling ocean: enough chop to launch from a crest without hammering the hull.
water = CreateWater(520, 520, 160)
SetWaterStyle("ocean")
SetWaterColor(8, 58, 86)
SetWaterWaves(4, 0.82)
SetGerstner(0, 0.94, 0.25, 0.30, 0.48, 24, 0.92)
SetGerstner(1, -0.20, 0.98, 0.24, 0.24, 12, 1.32)
SetGerstner(2, 0.62, -0.72, 0.18, 0.10, 5.5, 1.85)
SetGerstner(3, 0.18, 0.98, 0.12, 0.035, 2.7, 2.45)
SetWaterWind(0.90, 0.28, 0.62)
SetWaterFollow(True)
SetWaterLevel(0)
EnableWaterReflection(True)
EnableWaterRefraction(True)
SetWaterCaustics(water, True)

; Slalom markers give the open water a classic arcade course.
For i = 0 To 11
    marker = CreateSphere(10)
    SetScale(marker, 0.48, 0.48, 0.48)
    mx# = -7
    If i Mod 2 = 1 Then mx = 7
    mz# = 18 + i * 14
    SetPosition(marker, mx, WaterHeight(mx, mz) + 0.55, mz)
    If i Mod 2 = 0
        SetEntityColor(marker, 255, 92, 34)
    Else
        SetEntityColor(marker, 255, 218, 52)
    EndIf
    CreateBuoy(marker)
Next

; The physics body is the pivot; all visible parts are children.
jetski = CreatePivot()
SetPosition(jetski, 0, 1.35, 0)

hull = CreateCube(jetski)
SetScale(hull, 0.58, 0.25, 1.52)
SetPosition(hull, 0, -0.02, 0)
SetEntityColor(hull, 236, 48, 34)

deck = CreateCube(jetski)
SetScale(deck, 0.49, 0.13, 1.04)
SetPosition(deck, 0, 0.28, -0.10)
SetEntityColor(deck, 248, 82, 38)

nose = CreateCone(10, jetski)
SetScale(nose, 0.56, 0.72, 0.42)
SetPosition(nose, 0, 0.06, 1.30)
SetRotation(nose, 90, 0, 0)
SetEntityColor(nose, 248, 88, 38)

seat = CreateCube(jetski)
SetScale(seat, 0.35, 0.12, 0.52)
SetPosition(seat, 0, 0.48, -0.55)
SetEntityColor(seat, 28, 32, 38)

column = CreateCylinder(8, jetski)
SetScale(column, 0.08, 0.42, 0.08)
SetPosition(column, 0, 0.62, 0.38)
SetRotation(column, -16, 0, 0)
SetEntityColor(column, 42, 46, 50)

bars = CreateCube(jetski)
SetScale(bars, 0.48, 0.035, 0.035)
SetPosition(bars, 0, 0.98, 0.30)
SetEntityColor(bars, 28, 30, 34)

; Simple rider silhouette, leaning visually with the steering input.
rider = CreatePivot(jetski)
SetPosition(rider, 0, 0.52, -0.48)

body = CreateCylinder(10, rider)
SetScale(body, 0.25, 0.46, 0.21)
SetPosition(body, 0, 0.52, 0)
SetEntityColor(body, 28, 68, 172)

head = CreateSphere(10, rider)
SetScale(head, 0.19, 0.19, 0.19)
SetPosition(head, 0, 1.15, 0.06)
SetEntityColor(head, 255, 194, 142)

helmet = CreateSphere(10, rider)
SetScale(helmet, 0.205, 0.14, 0.205)
SetPosition(helmet, 0, 1.24, 0.04)
SetEntityColor(helmet, 244, 238, 224)

armL = CreateCube(rider)
SetScale(armL, 0.055, 0.055, 0.40)
SetPosition(armL, -0.24, 0.70, 0.40)
SetRotation(armL, -22, 9, 0)
SetEntityColor(armL, 255, 194, 142)

armR = CreateCube(rider)
SetScale(armR, 0.055, 0.055, 0.40)
SetPosition(armR, 0.24, 0.70, 0.40)
SetRotation(armR, -22, -9, 0)
SetEntityColor(armR, 255, 194, 142)

; White stern spray. Its rate follows speed.
spray = CreateEmitter(jetski)
SetPosition(spray, 0, -0.18, -1.48)
SetEmitterRate(spray, 0)
SetEmitterMax(spray, 90)
SetEmitterLife(spray, 0.55)
SetEmitterSpeed(spray, 4.2)
SetEmitterSize(spray, 0.10, 0.38)
SetEmitterColor(spray, 225, 244, 255, 0.82, 190, 226, 246, 0)
SetEmitterVelocity(spray, 0, 1.5, -3.8)
SetEmitterCone(spray, 30)
SetEmitterGravity(spray, 0, -5.5, 0)

CreateJetSkiController(jetski, 0.60, 0.35, 1.55, 380)
SetCCD(jetski, True)

camYaw# = 0
camPitch# = 13
steerSmooth# = 0
frames = 0

While 1
    frames = frames + 1
    If frames > 8 And KeyHit(KEY_ESCAPE) Then Exit
    dt# = DeltaTime()
    If dt <= 0 Then dt = 0.016
    If dt > 0.05 Then dt = 0.05

    camYaw = camYaw + MouseDeltaX() * 0.15
    camPitch = camPitch + MouseDeltaY() * 0.11
    If camPitch < -4 Then camPitch = -4
    If camPitch > 34 Then camPitch = 34
    MoveMouse(640, 360)

    throttle# = 0
    If KeyDown(KEY_W) Or KeyDown(KEY_UP) Then throttle = 1
    If KeyDown(KEY_S) Or KeyDown(KEY_DOWN) Then throttle = -0.42

    steer# = GetAxis(KEY_A, KEY_D)
    If KeyDown(KEY_LEFT) Then steer = -1
    If KeyDown(KEY_RIGHT) Then steer = 1
    steerSmooth = steerSmooth + (steer - steerSmooth) * 0.18

    UpdateJetSki(jetski, throttle, steerSmooth)

    speed# = BodyVelocity(jetski)
    SetEmitterRate(spray, Int(speed * 2.2))
    SetRotation(rider, 5, 0, -steerSmooth * 18)

    ; Recover if the player travels beyond the streamed course or sinks.
    wx# = EntityX(jetski)
    wz# = EntityZ(jetski)
    wy# = WaterHeight(wx, wz)
    If EntityY(jetski) < wy - 3 Or Abs(wx) > 220
        SetPosition(jetski, 0, WaterHeight(0, wz) + 1.2, wz)
        SetRotation(jetski, 0, EntityYaw(jetski), 0)
        SetVelocity(jetski, 0, 0, 0)
    EndIf

    chaseYaw# = EntityYaw(jetski) + camYaw
    CameraFollow(cam, jetski, 9.4, 2.9, 12, chaseYaw, camPitch)

    Cls
    UpdateWorld
    RenderWorld
    Text(20, 18, "JETSKI")
    Text(20, 42, "W / Up throttle    S reverse    A/D or arrows steer    mouse look    Esc quit")
    Text(20, 66, "Speed " + Int(speed * 1.94) + " knots")
    Flip
Wend
End
