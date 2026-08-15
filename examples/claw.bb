; Carriage is the parent. Pivots hang from it; Space rolls them shut.
; Prong bodies are kinematic and follow the parented meshes.

SetWindowTitle("BitShin BASIC — Jolt Claw Machine")
Graphics3D(1280, 720, 0, 2)
SetBuffer(BackBuffer())
SetCameraClsColor(30, 35, 45)
SetAmbientLight(70, 82, 100)

cam = CreateCamera()
SetPosition(cam, 0, 7.2, -13)
CameraRange(cam, 0.15, 4000)
PointEntity(cam, 0, 3.2, 0)

sun = CreateDirectionalLight()
SetLightDirection(sun, 40, 35, 0)
SetLightColor(sun, 255, 236, 200)
SetLightShadow(sun, True)
EnableShadows(True)
ShadowMapSize(1024)
SetShadowQuality(2, 4)
SetShadowBias(0.0018)
SetAmbientColor(61, 71, 92)

floor = CreateCube()
ScaleEntity(floor, 5.0, 0.5, 5.0)
SetPosition(floor, 0, 0, 0)
SetEntityColor(floor, 40, 45, 55)
CreateBodyBox(floor, 5.0, 0.5, 5.0, 0)

wallL = CreateCube()
ScaleEntity(wallL, 0.2, 4.0, 5.0)
SetPosition(wallL, -5.2, 4.0, 0)
SetEntityColor(wallL, 100, 150, 255)
SetEntityAlpha(wallL, 0.3)
CreateBodyBox(wallL, 0.2, 4.0, 5.0, 0)

wallR = CreateCube()
ScaleEntity(wallR, 0.2, 4.0, 5.0)
SetPosition(wallR, 5.2, 4.0, 0)
SetEntityColor(wallR, 100, 150, 255)
SetEntityAlpha(wallR, 0.3)
CreateBodyBox(wallR, 0.2, 4.0, 5.0, 0)

wallB = CreateCube()
ScaleEntity(wallB, 5.0, 4.0, 0.2)
SetPosition(wallB, 0, 4.0, 5.2)
SetEntityColor(wallB, 100, 150, 255)
SetEntityAlpha(wallB, 0.3)
CreateBodyBox(wallB, 5.0, 4.0, 0.2, 0)

wallF = CreateCube()
ScaleEntity(wallF, 5.0, 4.0, 0.2)
SetPosition(wallF, 0, 4.0, -5.2)
SetEntityColor(wallF, 100, 150, 255)
SetEntityAlpha(wallF, 0.22)
CreateBodyBox(wallF, 5.0, 4.0, 0.2, 0)

For i = 1 To 18
    prize = CreateCube()
    px# = Rnd(6.4) - 3.2
    py# = 1.2 + Rnd(3.6)
    pz# = Rnd(6.4) - 3.2
    ScaleEntity(prize, 0.30, 0.30, 0.30)
    SetPosition(prize, px, py, pz)
    SetEntityColor(prize, 50 + Rand(0, 205), 50 + Rand(0, 205), 50 + Rand(0, 205))
    prizeBody = CreateBodyBox(prize, 0.30, 0.30, 0.30, 2, 0.10)
    SetRestitution(prizeBody, 0.02)
    SetFriction(prizeBody, 2.4)
    SetLinearDamping(prizeBody, 0.35)
    SetAngularDamping(prizeBody, 0.55)
    SetBodyCCD(prizeBody, 1)
Next

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
SetPosition(carriage, 0, carY, 0)
SetEntityColor(carriage, 200, 50, 50)
carriageBody = CreateBodyBox(carriage, carHX, carHY, carHZ, 1)
SetFriction(carriageBody, 3.2)
SetRestitution(carriageBody, 0)

pivotL = CreatePivot(carriage)
SetPosition(pivotL, prongLX, -carHY, 0)
prongL = CreateBox(prongHX * 2, prongHY * 2, prongHZ * 2)
EntityParent(prongL, pivotL)
SetPosition(prongL, 0, -prongHY, 0)
SetEntityColor(prongL, 180, 180, 180)
prongLBody = CreateBodyBox(prongL, prongHX, prongHY, prongHZ, 1)
SetFriction(prongLBody, 3.2)
SetRestitution(prongLBody, 0)
SetBodyCCD(prongLBody, 1)

pivotR = CreatePivot(carriage)
SetPosition(pivotR, prongRX, -carHY, 0)
prongR = CreateBox(prongHX * 2, prongHY * 2, prongHZ * 2)
EntityParent(prongR, pivotR)
SetPosition(prongR, 0, -prongHY, 0)
SetEntityColor(prongR, 180, 180, 180)
prongRBody = CreateBodyBox(prongR, prongHX, prongHY, prongHZ, 1)
SetFriction(prongRBody, 3.2)
SetRestitution(prongRBody, 0)
SetBodyCCD(prongRBody, 1)

DisableBodyCollision(carriageBody, prongLBody)
DisableBodyCollision(carriageBody, prongRBody)
DisableBodyCollision(prongLBody, prongRBody)

Print "Physics backend:", GetPhysicsBackend()

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
