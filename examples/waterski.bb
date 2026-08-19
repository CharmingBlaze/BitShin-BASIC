; DarkBASIC-style waterski: tow boat, choppy Gerstner, carve on the edge.

SetWindowTitle("BitShin BASIC — Waterski")
Graphics3D(1280, 720, 0, 2)
SetBuffer(BackBuffer())
SetCameraClsColor(72, 148, 210)
SetAmbientLight(78, 92, 118)
HidePointer()
MoveMouse(640, 360)

cam = CreateCamera()
SetCameraRange(cam, 0.2, 2800)
sky = CreateSkyBox()
CameraFogMode(cam, 1)
CameraFogColor(cam, 150, 188, 222)
CameraFogRange(cam, 120, 520)
SetWeather("clear")
SetWeatherIntensity(0.7)

sun = CreateDirectionalLight()
SetLightDirection(sun, 52, 38, 0)
SetLightColor(sun, 255, 236, 198)

water = CreateWater(480, 480, 160)
SetWaterStyle("ocean")
SetWaterColor(8, 52, 78)
SetWaterWaves(4, 0.82)
SetGerstner(0, 0.92, 0.28, 0.30, 0.50, 22, 1.05)
SetGerstner(1, -0.18, 0.98, 0.24, 0.26, 11, 1.45)
SetGerstner(2, 0.62, -0.72, 0.18, 0.10, 5.5, 1.9)
SetGerstner(3, 0.22, 0.97, 0.12, 0.035, 2.4, 2.6)
SetWaterWind(0.88, 0.32, 0.68)
SetWaterFollow(True)
SetWaterLevel(0)
EnableWaterReflection(True)
EnableWaterRefraction(True)
SetWaterCaustics(water, True)

; Marker buoys so the chop reads in third person.
For i = 0 To 5
    b = CreateSphere(8)
    SetScale(b, 0.55, 0.55, 0.55)
    SetPosition(b, -18 + i * 8, 0.8, 14 + (i Mod 2) * 10)
    SetEntityColor(b, 220, 70 - i * 8, 48)
    CreateBuoy(b)
Next

; Tow boat (player skis behind this).
boat = CreatePivot()
SetPosition(boat, 0, 1.15, 16)

hull = CreateCube(boat)
SetScale(hull, 1.35, 0.38, 3.1)
SetPosition(hull, 0, 0, 0)
SetEntityColor(hull, 28, 92, 168)

bow = CreateCone(8, boat)
SetScale(bow, 1.2, 1.1, 0.55)
SetPosition(bow, 0, 0.05, 2.55)
SetRotation(bow, 90, 0, 0)
SetEntityColor(bow, 24, 82, 152)

cabin = CreateCube(boat)
SetScale(cabin, 0.7, 0.45, 0.85)
SetPosition(cabin, 0, 0.62, -0.35)
SetEntityColor(cabin, 236, 236, 242)

wind = CreateCube(boat)
SetScale(wind, 0.68, 0.28, 0.06)
SetPosition(wind, 0, 0.95, 0.4)
SetEntityColor(wind, 120, 190, 230)
SetEntityAlpha(wind, 0.45)

CreateBoatController(boat)
SetLinearDamping(boat, 1.5)
stern = CreatePivot(boat)
SetPosition(stern, 0, 0.2, -2.8)

; Skier + two boards. Physics lives on the pivot.
skier = CreatePivot()
SetPosition(skier, 0, 0.55, 2)

skiL = CreateCube(skier)
SetScale(skiL, 0.11, 0.045, 0.95)
SetPosition(skiL, -0.22, -0.02, 0.1)
SetEntityColor(skiL, 240, 236, 220)

skiR = CreateCube(skier)
SetScale(skiR, 0.11, 0.045, 0.95)
SetPosition(skiR, 0.22, -0.02, 0.1)
SetEntityColor(skiR, 240, 236, 220)

torso = CreateCylinder(10, skier)
SetScale(torso, 0.22, 0.42, 0.18)
SetPosition(torso, 0, 0.72, -0.05)
SetEntityColor(torso, 210, 48, 42)

head = CreateSphere(9, skier)
SetScale(head, 0.18, 0.18, 0.18)
SetPosition(head, 0, 1.28, -0.02)
SetEntityColor(head, 255, 198, 148)

armL = CreateCube(skier)
SetScale(armL, 0.06, 0.06, 0.38)
SetPosition(armL, -0.28, 0.95, 0.28)
SetRotation(armL, -18, 12, 0)
SetEntityColor(armL, 255, 198, 148)

armR = CreateCube(skier)
SetScale(armR, 0.06, 0.06, 0.38)
SetPosition(armR, 0.28, 0.95, 0.28)
SetRotation(armR, -18, -12, 0)
SetEntityColor(armR, 255, 198, 148)

handle = CreateCube(skier)
SetScale(handle, 0.42, 0.035, 0.035)
SetPosition(handle, 0, 0.92, 0.58)
SetEntityColor(handle, 40, 40, 44)

CreateWaterSkiController(skier)
SetLinearDamping(skier, 0.45)

; A real Jolt rope: boat stern anchor -> skier handle anchor.
towRope = CreateRopeAnchored(boat, skier, 0, 0.2, -2.8, 0, 0.92, 0.58, 13.5, 14, 0.028)
SetRopeColor(towRope, 226, 204, 132)
SetRopeMass(towRope, 0.09)
SetRopeDamping(towRope, 0.24)
SetRopeStrength(towRope, 4000, 500, 3500)

camYaw# = 0
camPitch# = 16
edgeSmo# = 0
frames = 0

While 1
    frames = frames + 1
    If frames > 8 And KeyHit(KEY_ESCAPE) Then Exit
    dt# = DeltaTime()
    If dt <= 0 Then dt = 0.016
    If dt > 0.05 Then dt = 0.05

    camYaw = camYaw + MouseDeltaX() * 0.16
    camPitch = camPitch + MouseDeltaY() * 0.12
    If KeyDown(KEY_LEFT) Then camYaw = camYaw - 70 * dt
    If KeyDown(KEY_RIGHT) Then camYaw = camYaw + 70 * dt
    If camPitch > 42 Then camPitch = 42
    If camPitch < -8 Then camPitch = -8
    MoveMouse(640, 360)

    pull# = 0.65
    If KeyDown(KEY_W) Or KeyDown(KEY_UP) Then pull = 1.0
    If KeyDown(KEY_S) Or KeyDown(KEY_DOWN) Then pull = 0.2

    edge# = GetAxis(KEY_A, KEY_D)
    edgeSmo = edgeSmo + (edge - edgeSmo) * 0.18

    boatTh# = 0.82
    If pull > 0.85 Then boatTh = 1.0
    boatSteer# = Sin(MilliSecs() * 0.00035) * 0.18
    UpdateBoat(boat, boatTh, boatSteer)

    UpdateWaterSki(skier, pull, edgeSmo, boat)

    ; The rope constraint transfers the boat's pull. UpdateWaterSki detects the
    ; attached physical rope and does not add the old spring approximation.
    sx# = EntityX(skier)
    sy# = EntityY(skier) + 0.9
    sz# = EntityZ(skier)
    rlen# = EntityDistance(skier, boat)

    SetRotation(torso, 10 + (1 - pull) * 16, 0, -edgeSmo * 26)

    wy# = WaterHeight(sx, sz)
    If sy < wy - 2.8 Or rlen > 42 Then
        SetPosition(skier, EntityX(boat), wy + 0.55, EntityZ(boat) - 12)
        SetVelocity(skier, 0, 0, 0)
        SetBodyRotation(skier, 0, EntityYaw(boat), 0)
        ResetRope(towRope)
    EndIf

    chaseYaw# = EntityYaw(skier) + camYaw
    CameraFollow(cam, skier, 10.5, 3.15, 11, chaseYaw, camPitch)

    Cls
    UpdateWorld
    RenderWorld
    spd# = BodyVelocity(skier)
    Text(18, 16, "Waterski")
    Text(18, 40, "W pull   A/D carve   S sit back   mouse look   Esc quit")
    Text(18, 64, "Speed " + Int(spd) + "    rope " + Int(RopeTension(towRope) * 100) + "%    chop " + Str(WaterHeight(sx, sz)))
    Flip
Wend
End
