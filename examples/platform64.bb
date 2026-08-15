; Original 64-style 3D platformer — no trademarked names or assets.

SetWindowTitle("Platform 64")
Graphics3D(1280, 720, 0, 2)
SetBuffer(BackBuffer())
SetCameraClsColor(70, 140, 210)
SetAmbientLight(92, 108, 138)
HidePointer()
MoveMouse(640, 360)

cam = CreateCamera()
SetCameraRange(cam, 0.1, 4000)
sky = CreateSkyBox()
CameraFogMode(cam, 1)
CameraFogColor(cam, 145, 175, 205)
CameraFogRange(cam, 45, 220)
SetWeather("clear")
SetWeatherIntensity(0.85)

EnableShadows True
ShadowCascades 2
ShadowMapSize 2048
SetShadowBias 0.0018
SetShadowQuality(2, 4)

sun = CreateDirectionalLight()
SetLightDirection sun, 55, 40, 0
SetLightColor sun, 255, 235, 200
SetLightShadow sun, True

texGrass = LoadTexture("assets/grass.png")
texStone = LoadTexture("assets/stone.png")
texCoin = LoadTexture("assets/coin.png")
ScaleTexture(texGrass, 8, 8)
ScaleTexture(texStone, 2, 2)
sndCoin = LoadSound("assets/coin.wav")

player = CreatePivot()
body = CreateCylinder(10, player)
SetScale(body, 0.45, 0.55, 0.45)
SetPosition(body, 0, 0.7, 0)
SetEntityColor(body, 40, 170, 150)
EntityShininess(body, 0.08)
EntitySpecular(body, 36, 48, 42)
head = CreateSphere(10, player)
SetScale(head, 0.38, 0.38, 0.38)
SetPosition(head, 0, 1.45, 0)
SetEntityColor(head, 255, 196, 140)
EntityShininess(head, 0.1)
EntitySpecular(head, 40, 32, 28)

Dim padX(20)
Dim padZ(20)
Dim padW(20)
Dim padD(20)
Dim padY(20)
nPad = 0

Function AddPad(x#, y#, z#, hx#, hz#, r, g, b, tex, shine#, sr, sg, sb)
    nPad = nPad + 1
    padX(nPad) = x
    padZ(nPad) = z
    padW(nPad) = hx
    padD(nPad) = hz
    padY(nPad) = y + 0.25
    block = CreateCube()
    SetScale(block, hx, 0.25, hz)
    SetPosition(block, x, y, z)
    SetEntityColor(block, r, g, b)
    If tex Then EntityTexture(block, tex)
    EntityShininess(block, shine)
    EntitySpecular(block, sr, sg, sb)
End Function

AddPad(0, 0, 0, 18, 18, 46, 130, 70, texGrass, 0.03, 16, 22, 14)
AddPad(10, 1.2, 8, 3, 3, 200, 160, 70, texStone, 0.11, 48, 40, 28)
AddPad(16, 2.6, 12, 2.4, 2.4, 80, 150, 200, texStone, 0.11, 48, 40, 28)
AddPad(-8, 1.6, 10, 2.8, 2.8, 180, 90, 70, texStone, 0.11, 48, 40, 28)
AddPad(-4, 3.2, 16, 3.2, 3.2, 90, 80, 160, texStone, 0.11, 48, 40, 28)
For i = 0 To 4
    AddPad(-12 + i * 1.1, 0.35 + i * 0.55, 4 + i * 0.2, 0.7, 0.9, 170, 120, 80, texStone, 0.12, 52, 42, 30)
Next
AddPad(4, 4.4, 20, 4, 3, 50, 160, 120, texStone, 0.11, 48, 40, 28)

ring = CreateTorus(1.6, 0.18, 16, 24)
SetPosition(ring, -16, 3.2, -10)
SetEntityColor(ring, 220, 170, 70)
EntityShininess(ring, 0.22)
EntitySpecular(ring, 70, 58, 36)
spire = CreateCone(8, 0)
SetScale(spire, 0.7, 2.2, 0.7)
SetPosition(spire, 20, 2.2, -6)
SetEntityColor(spire, 90, 140, 170)
EntityShininess(spire, 0.08)
EntitySpecular(spire, 30, 34, 40)
marker = CreateCapsule(0.28, 0.9, 10)
SetPosition(marker, -14, 1.4, 8)
SetEntityColor(marker, 200, 90, 80)
EntityShininess(marker, 0.1)
EntitySpecular(marker, 44, 28, 24)

Dim coinX(8)
Dim coinY(8)
Dim coinZ(8)
Dim coinE(8)
Dim coinOn(8)
coinX(1) = 10 : coinY(1) = 2.4 : coinZ(1) = 8
coinX(2) = 16 : coinY(2) = 3.8 : coinZ(2) = 12
coinX(3) = -8 : coinY(3) = 2.8 : coinZ(3) = 10
coinX(4) = -4 : coinY(4) = 4.4 : coinZ(4) = 16
coinX(5) = 4 : coinY(5) = 5.6 : coinZ(5) = 20
coinX(6) = -10 : coinY(6) = 2.2 : coinZ(6) = 5
coinX(7) = 2 : coinY(7) = 1.2 : coinZ(7) = 6
coinX(8) = 0 : coinY(8) = 1.2 : coinZ(8) = -6
For i = 1 To 8
    coinE(i) = CreateSphere(8)
    SetScale(coinE(i), 0.28, 0.28, 0.28)
    SetPosition(coinE(i), coinX(i), coinY(i), coinZ(i))
    SetEntityColor(coinE(i), 255, 220, 50)
    EntityTexture(coinE(i), texCoin)
    EntityShininess(coinE(i), 0.28)
    EntitySpecular(coinE(i), 80, 70, 36)
    coinOn(i) = 1
Next

px# = 0
py# = 0.5
pz# = -4
vx# = 0
vz# = 0
vy# = 0
yaw# = 0
camYaw# = 0
camPitch# = 12
coins = 0
grounded = 1
SetPosition(player, px, py, pz)
SetPosition(cam, px, py + 4.2, pz - 8)

Function AngWrap#(a#)
    While a > 180
        a = a - 360
    Wend
    While a < -180
        a = a + 360
    Wend
    Return a
End Function

frames = 0
While 1
    frames = frames + 1
    If frames > 8 And KeyHit(KEY_ESCAPE) Then Exit
    dt# = DeltaTime()
    If dt <= 0 Then dt = 0.016
    If dt > 0.05 Then dt = 0.05

    camYaw = camYaw + MouseDeltaX() * 0.18
    camPitch = camPitch + MouseDeltaY() * 0.14
    If KeyDown(KEY_LEFT) Then camYaw = camYaw - 90 * dt
    If KeyDown(KEY_RIGHT) Then camYaw = camYaw + 90 * dt
    If KeyDown(KEY_UP) Then camPitch = camPitch - 70 * dt
    If KeyDown(KEY_DOWN) Then camPitch = camPitch + 70 * dt
    If camPitch > 48 Then camPitch = 48
    If camPitch < -18 Then camPitch = -18
    MoveMouse(640, 360)

    wishX# = 0
    wishZ# = 0
    If KeyDown(KEY_W) Then
        wishX = wishX + Sin(camYaw)
        wishZ = wishZ + Cos(camYaw)
    EndIf
    If KeyDown(KEY_S) Then
        wishX = wishX - Sin(camYaw)
        wishZ = wishZ - Cos(camYaw)
    EndIf
    If KeyDown(KEY_A) Then
        wishX = wishX + Sin(camYaw - 90)
        wishZ = wishZ + Cos(camYaw - 90)
    EndIf
    If KeyDown(KEY_D) Then
        wishX = wishX + Sin(camYaw + 90)
        wishZ = wishZ + Cos(camYaw + 90)
    EndIf
    wishLen# = Sqr(wishX * wishX + wishZ * wishZ)
    If wishLen > 0.001 Then
        wishX = wishX / wishLen
        wishZ = wishZ / wishLen
    EndIf

    maxSpd# = 8.4
    If grounded Then
        fric# = 16
        acc# = 42
    Else
        fric# = 1.1
        acc# = 11
    EndIf
    spd# = Sqr(vx * vx + vz * vz)
    If spd > 0.001 Then
        drop# = fric * dt
        nspd# = spd - drop
        If nspd < 0 Then nspd = 0
        vx = vx * (nspd / spd)
        vz = vz * (nspd / spd)
    EndIf
    If wishLen > 0.001 Then
        vx = vx + wishX * acc * dt
        vz = vz + wishZ * acc * dt
        hsp# = Sqr(vx * vx + vz * vz)
        cap# = maxSpd
        If grounded = 0 And hsp > maxSpd Then cap = hsp
        If grounded And hsp > maxSpd Then
            vx = vx * (maxSpd / hsp)
            vz = vz * (maxSpd / hsp)
        ElseIf grounded = 0 And hsp > cap Then
            vx = vx * (cap / hsp)
            vz = vz * (cap / hsp)
        EndIf
    EndIf

    px = px + vx * dt
    pz = pz + vz * dt

    faceSpd# = Sqr(vx * vx + vz * vz)
    If faceSpd > 0.45 Then
        want# = ATan2(vx, vz)
        diff# = AngWrap(want - yaw)
        turn# = 640 * dt
        If grounded = 0 Then turn = 150 * dt
        If Abs(diff) <= turn Then
            yaw = want
        ElseIf diff > 0 Then
            yaw = yaw + turn
        Else
            yaw = yaw - turn
        EndIf
    EndIf

    If KeyHit(KEY_1) Then SetWeather("clear")
    If KeyHit(KEY_2) Then SetWeather("rain")
    If KeyHit(KEY_3) Then SetWeather("snow")
    If KeyHit(KEY_4) Then SetWeather("fog")
    If KeyHit(KEY_5) Then SetWeather("storm")

    If KeyHit(KEY_SPACE) And grounded Then vy = 11.2
    If grounded = 0 And KeyDown(KEY_SPACE) = 0 And vy > 2 Then vy = vy - 18 * dt
    grav# = 30
    If vy < 0 Then grav = 38
    vy = vy - grav * dt
    py = py + vy * dt

    grounded = 0
    best# = -999
    For i = 1 To nPad
        If Abs(px - padX(i)) < padW(i) And Abs(pz - padZ(i)) < padD(i) Then
            top# = padY(i)
            If py <= top + 0.45 And py >= top - 0.85 And vy <= 0.35 Then
                If top > best Then best = top
            EndIf
        EndIf
    Next
    If best > -900 Then
        py = best
        vy = 0
        grounded = 1
    EndIf
    If py < -8 Then
        px = 0 : py = 0.5 : pz = -4 : vy = 0 : vx = 0 : vz = 0
    EndIf

    SetPosition(player, px, py, pz)
    SetRotation(player, 0, yaw, 0)

    bobT# = MilliSecs() * 0.004
    For i = 1 To 8
        If coinOn(i) Then
            bob# = Sin(bobT * 57.3 + i * 40) * 0.16
            SetPosition(coinE(i), coinX(i), coinY(i) + bob, coinZ(i))
            TurnEntity(coinE(i), 0, 140 * dt, 0)
            If Dist(px, pz, coinX(i), coinZ(i)) < 1.1 And Abs(py + 0.8 - coinY(i)) < 1.2 Then
                If coinE(i) Then HideEntity(coinE(i))
                coinOn(i) = 0
                coins = coins + 1
                If sndCoin Then PlaySound(sndCoin)
            EndIf
        EndIf
    Next

    CameraFollow(cam, player, 8.2, 3.05, 8.5, camYaw, camPitch)

    Cls
    UpdateWorld
    RenderWorld
    Text(20, 16, "Platform 64")
    Text(20, 42, "Coins  " + coins + " / 8     WASD move   Space jump   Mouse / arrows look   Esc quit")
    Text(20, 68, "Weather  " + Weather$() + "    1 clear  2 rain  3 snow  4 fog  5 storm")
    If coins = 8 Then Text(20, 94, "All coins! Nice run.")
    Flip
Wend
End
