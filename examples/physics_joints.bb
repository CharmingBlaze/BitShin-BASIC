; Hinged door — sits in the jamb opening; Space nudges it open.
; Esc quits after a few frames.

SetWindowTitle("BitShin BASIC — Physics joints")
Graphics3D(960, 600, 0, 2)
SetBuffer(BackBuffer())
SetCameraClsColor(70, 120, 190)
SetAmbientLight(70, 82, 100)

; Camera
cam = CreateCamera()
cam.Position([0, 2.4, -10])
CameraRange(cam, 0.15, 4000)
cam.Point(0, 1.5, 0)

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

; Ground (cube Scale = half-extents; top at y=0)
ground = CreateCube().Scale(10, 0.1, 10).Position([0, -0.1, 0]).Color(62, 92, 58)
CreateBodyBox(ground, 10, 0.1, 10, 0)

; Left jamb AABB X [-3.55, -2.85]. Right jamb AABB X [2.05, 2.75].
; Opening X (-2.85, 2.05).
leftJamb = CreateCube().Scale(0.35, 1.80, 0.40).Position([-3.20, 1.80, 0]).Color(150, 156, 170)
leftBody = CreateBodyBox(leftJamb, 0.35, 1.80, 0.40, 0)

rightJamb = CreateCube().Scale(0.35, 1.80, 0.40).Position([2.40, 1.80, 0]).Color(150, 156, 170)
rightBody = CreateBodyBox(rightJamb, 0.35, 1.80, 0.40, 0)

; Door LEFT edge = hinge. halfX=1.05 → door X [-2.40, -0.30].
; Gap to left jamb: 0.45 m. Gap to right jamb: 2.35 m. Bottom y=0.17.
doorHalfX# = 1.05
doorHalfY# = 1.55
doorHalfZ# = 0.06
hingeX# = -2.40
hingeY# = 1.72
hingeZ# = 0
door = CreateCube().Scale(doorHalfX, doorHalfY, doorHalfZ).Position([hingeX + doorHalfX, hingeY, hingeZ]).Color(230, 150, 70)
doorBody = CreateBodyBox(door, doorHalfX, doorHalfY, doorHalfZ, 20)

SetRestitution(doorBody, 0)
SetFriction(doorBody, 0.35)
SetLinearDamping(doorBody, 1.2)
SetAngularDamping(doorBody, 2.0)

; World hinge on the door's left edge
doorHinge = CreateHingeJoint(0, doorBody, hingeX, hingeY, hingeZ, 0, 1, 0)
SetHingeLimits(doorHinge, 0, 90)
SetHingeFriction(doorHinge, 18)
DisableBodyCollision(doorBody, leftBody)
DisableBodyCollision(doorBody, rightBody)

Print "Physics backend:", GetPhysicsBackend()
Print "Hinge id:", doorHinge
Print "layout leftJambX", -3.20, "rightFace", -2.85, "hingeX", hingeX, "doorRight", hingeX + doorHalfX * 2

frames = 0
While 1
    frames = frames + 1
    If frames = 12 Then ApplyImpulse(doorBody, 0, 0, 2.4)
    If KeyHit(KEY_SPACE) Then ApplyImpulse(doorBody, 0, 0, 2.4)
    UpdateWorld()
    RenderWorld()
    Text(16, 16, "Space = nudge door")
    Text(16, 36, "Esc = quit")
    Text(16, 56, "backend " + GetPhysicsBackend() + "  hinge " + doorHinge)
    Flip()
    If frames > 8 And KeyHit(KEY_ESCAPE) Then End
Wend
End
