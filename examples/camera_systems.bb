; BitShin BASIC — Modern Camera Systems Showcase Demo
SetWindowTitle("BitShin BASIC — Modern Camera Systems")
Graphics3D(1280, 720, 0, 2)
SetBuffer(BackBuffer())
SetCameraClsColor(40, 50, 65)
SetAmbientLight(60, 70, 80)

cam = CreateCamera()
SetCameraRange(cam, 0.1, 1000)

sun = CreateDirectionalLight()
SetLightDirection(sun, 50, 35, 0)
SetLightColor(sun, 255, 235, 200)
SetLightShadow(sun, True)
EnableShadows(True)

; Scene
ground = CreateCube()
SetScale(ground, 60, 0.25, 60)
SetPosition(ground, 0, 0, 0)
SetEntityColor(ground, 50, 80, 60)
CreateRigidBodyBox(ground, 60, 0.25, 60, 0)

target = CreateCube()
SetScale(target, 1.5, 1.5, 1.5)
SetPosition(target, 0, 1.5, 0)
SetEntityColor(target, 220, 80, 60)

camMode = 1
orbitYaw# = 0.0
rtsX# = 0.0
rtsZ# = 0.0
rtsZoom# = 20.0

frames = 0
While 1
    frames = frames + 1
    If frames > 8 And KeyHit(KEY_ESCAPE) Then Exit
    dt# = DeltaTime()

    If KeyHit(KEY_1) Then camMode = 1
    If KeyHit(KEY_2) Then camMode = 2
    If KeyHit(KEY_3) Then camMode = 3
    If KeyHit(KEY_SPACE) Then AddCameraShake(0.6)

    ; Move target
    tx# = EntityX(target) + GetAxis(KEY_A, KEY_D) * 12.0 * dt
    tz# = EntityZ(target) + GetAxis(KEY_S, KEY_W) * 12.0 * dt
    SetPosition(target, tx, 1.5, tz)

    If camMode = 1 Then
        ; Mode 1: SpringArm Follow Camera
        CameraSpringArm(cam, target, 1.5, 8.0, 0.3, 12.0, 0, 20)
    ElseIf camMode = 2 Then
        ; Mode 2: Orbital Camera
        orbitYaw = orbitYaw + 45.0 * dt
        CameraOrbit(cam, target, 12.0, 3.5, orbitYaw, 25.0)
    ElseIf camMode = 3 Then
        ; Mode 3: Strategy / RTS Camera
        rtsX = tx
        rtsZ = tz
        CameraRTS(cam, rtsX, rtsZ, rtsZoom, 55.0, 0)
    EndIf

    ; Process procedural camera shake
    UpdateCameraShake(cam)

    Cls
    UpdateWorld
    RenderWorld

    Text(20, 20, "Modern Camera Systems Showcase")
    Text(20, 44, "1: Spring-Arm Anti-Clip | 2: Smooth Orbit | 3: RTS Strategy | Space: Trigger Camera Shake | Esc: Quit")
    modeName$ = "Spring-Arm"
    If camMode = 2 Then modeName$ = "Smooth Orbit"
    If camMode = 3 Then modeName$ = "RTS Strategy"
    Text(20, 68, "Current Mode: " + modeName$ + " | WASD: Move Red Target")
    Flip
Wend
End
