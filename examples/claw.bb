; Claw machine — carriage + hinged prongs grab falling prizes.
; WASD move, Up/Down raise/lower, hold Space to close. Esc quits.

SetWindowTitle("BitShin BASIC — Jolt Claw Machine")
Graphics3D(1280, 720, 0, 2)
SetBuffer(BackBuffer())
SetCameraClsColor(30, 35, 45)
SetAmbientLight(70, 82, 100)

; Camera
cam = CreateCamera()
cam.Position([0, 7.2, -13])
CameraRange(cam, 0.15, 4000)
cam.Point(0, 3.2, 0)

; Sun + shadows
sun = CreateDirectionalLight()
SetLightDirection(sun, 40, 35, 0)
SetLightColor(sun, 255, 236, 200)
SetLightShadow(sun, True)
EnableShadows(True)
ShadowMapSize(1024)
SetShadowQuality(0, 4)
SetShadowBias(0.0018)
SetAmbientColor(61, 71, 92)

; Cabinet floor + glass walls
floor = CreateCube().Scale(5.0, 0.5, 5.0).Position([0, 0, 0]).Color(40, 45, 55)
CreateBodyBox(floor, 5.0, 0.5, 5.0, 0)

wallL = CreateCube().Scale(0.2, 4.0, 5.0).Position([-5.2, 4.0, 0]).Color(100, 150, 255).Alpha(0.3)
CreateBodyBox(wallL, 0.2, 4.0, 5.0, 0)

wallR = CreateCube().Scale(0.2, 4.0, 5.0).Position([5.2, 4.0, 0]).Color(100, 150, 255).Alpha(0.3)
CreateBodyBox(wallR, 0.2, 4.0, 5.0, 0)

wallB = CreateCube().Scale(5.0, 4.0, 0.2).Position([0, 4.0, 5.2]).Color(100, 150, 255).Alpha(0.3)
CreateBodyBox(wallB, 5.0, 4.0, 0.2, 0)

wallF = CreateCube().Scale(5.0, 4.0, 0.2).Position([0, 4.0, -5.2]).Color(100, 150, 255).Alpha(0.22)
CreateBodyBox(wallF, 5.0, 4.0, 0.2, 0)

; Prizes
For i = 1 To 18
    prize = CreateCube()
    px# = Rnd(6.4) - 3.2
    py# = 1.2 + Rnd(3.6)
    pz# = Rnd(6.4) - 3.2
    prize.Scale(0.30, 0.30, 0.30)
    prize.Position([px, py, pz])
    prize.Color(50 + Rand(0, 205), 50 + Rand(0, 205), 50 + Rand(0, 205))
    prizeBody = CreateBodyBox(prize, 0.30, 0.30, 0.30, 2, 0.10)
    SetRestitution(prizeBody, 0.02)
    SetFriction(prizeBody, 2.4)
    SetLinearDamping(prizeBody, 0.35)
    SetAngularDamping(prizeBody, 0.55)
    SetBodyCCD(prizeBody, 1)
Next

; Carriage + prong layout
carHX# = 0.42
carHY# = 0.16
carHZ# = 0.42
carY# = 8.4
gap# = 0.08
prongHX# = 0.18
prongHY# = 1.05
prongHZ# = 0.28
prongLX# = -(carHX + gap + prongHX)
prongRX# = carHX + gap + prongHX

carriage = CreateBox(carHX * 2, carHY * 2, carHZ * 2)
carriage.Position([0, carY, 0])
carriage.Color(200, 50, 50)
carriageBody = CreateBodyBox(carriage, carHX, carHY, carHZ, 1)
SetFriction(carriageBody, 3.2)
SetRestitution(carriageBody, 0)

; Left prong (pivot hangs from carriage; body follows parented mesh)
pivotL = CreatePivot(carriage)
pivotL.Position([prongLX, -carHY, 0])
prongL = CreateBox(prongHX * 2, prongHY * 2, prongHZ * 2)
prongL.Parent(pivotL)
prongL.Position([0, -prongHY, 0])
prongL.Color(180, 180, 180)
prongLBody = CreateBodyBox(prongL, prongHX, prongHY, prongHZ, 1)
SetFriction(prongLBody, 3.2)
SetRestitution(prongLBody, 0)
SetBodyCCD(prongLBody, 1)

; Right prong
pivotR = CreatePivot(carriage)
pivotR.Position([prongRX, -carHY, 0])
prongR = CreateBox(prongHX * 2, prongHY * 2, prongHZ * 2)
prongR.Parent(pivotR)
prongR.Position([0, -prongHY, 0])
prongR.Color(180, 180, 180)
prongRBody = CreateBodyBox(prongR, prongHX, prongHY, prongHZ, 1)
SetFriction(prongRBody, 3.2)
SetRestitution(prongRBody, 0)
SetBodyCCD(prongRBody, 1)

DisableBodyCollision(carriageBody, prongLBody)
DisableBodyCollision(carriageBody, prongRBody)
DisableBodyCollision(prongLBody, prongRBody)

Print "Physics backend:", GetPhysicsBackend()

; Travel limits
minX# = -3.7
maxX# = 3.7
minZ# = -3.7
maxZ# = 3.7
minY# = 3.6
maxY# = 9.2
grip# = 0

While True
    vx# = 0
    vy# = 0
    vz# = 0
    If KeyDown(KEY_W) Then vz = 3.5
    If KeyDown(KEY_S) Then vz = -3.5
    If KeyDown(KEY_A) Then vx = -3.5
    If KeyDown(KEY_D) Then vx = 3.5
    If KeyDown(KEY_DOWN) Then vy = -2.6
    If KeyDown(KEY_UP) Then vy = 2.6

    cx# = EntityX(carriage)
    cy# = EntityY(carriage)
    cz# = EntityZ(carriage)
    If cx <= minX And vx < 0 Then vx = 0
    If cx >= maxX And vx > 0 Then vx = 0
    If cz <= minZ And vz < 0 Then vz = 0
    If cz >= maxZ And vz > 0 Then vz = 0
    If cy <= minY And vy < 0 Then vy = 0
    If cy >= maxY And vy > 0 Then vy = 0

    SetLinearVelocity(carriageBody, vx, vy, vz)

    want# = 0
    ; ~19deg meets the tips; stay under that so the claws form a V, not an X.
    If KeyDown(KEY_SPACE) Then want = 18
    grip = grip + (want - grip) * 0.18
    RotateEntity(pivotL, 0, 0, -grip)
    RotateEntity(pivotR, 0, 0, grip)

    UpdateWorld()
    RenderWorld()
    Text(16, 16, "WASD = move claw X/Z")
    Text(16, 36, "Up/Down = raise / lower")
    Text(16, 56, "Hold Space = close   release = open")
    Text(16, 76, "Esc = quit   backend " + GetPhysicsBackend())
    Flip()
    If KeyHit(KEY_ESCAPE) Then End
Wend
End
