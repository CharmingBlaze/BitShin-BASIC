; Mesh primitive showcase — Esc quits

Graphics3D(960, 600)
SetBuffer(BackBuffer())
SetWindowTitle("BitShin BASIC — primitives")
SetCameraClsColor(18, 22, 32)
SetAmbientLight(50, 55, 70)

cam = CreateCamera()
SetPosition(cam, 0, 3, -12)
CreateSkyBox()
CameraFogMode(cam, 1)
CameraFogColor(cam, 18, 22, 32)
CameraFogRange(cam, 14, 40)

sun = CreateDirectionalLight()
SetLightDirection(sun, 40, 30, 0)
SetLightColor(sun, 255, 240, 210)

ground = CreatePlane(28, 28)
SetPosition(ground, 0, 0, 5)
SetEntityColor(ground, 32, 38, 48)

box = CreateBox(1.4, 0.8, 1.1)
SetPosition(box, -4, 0.6, 4)
SetEntityColor(box, 220, 90, 80)

sph = CreateSphere(1, 16, 0)
SetPosition(sph, -2, 0.8, 4)
SetEntityColor(sph, 80, 180, 220)

cyl = CreateCylinder(0.45, 1.6, 14)
SetPosition(cyl, 0, 0.8, 4)
SetEntityColor(cyl, 90, 200, 120)

cone = CreateCone(0.55, 1.6, 12)
SetPosition(cone, 2, 0.8, 4)
SetEntityColor(cone, 230, 190, 70)

tor = CreateTorus(0.7, 0.22, 14, 20)
SetPosition(tor, 4, 0.9, 4)
SetEntityColor(tor, 200, 120, 220)

cap = CreateCapsule(0.35, 0.9, 12)
SetPosition(cap, -3, 0.9, 7)
SetEntityColor(cap, 240, 240, 240)

pyr = CreatePyramid(1.4)
SetPosition(pyr, -1, 0.7, 7)
SetEntityColor(pyr, 70, 140, 200)

wed = CreateWedge(1.4, 0.8, 1.2)
SetPosition(wed, 1.4, 0.4, 7)
SetEntityColor(wed, 180, 100, 70)

tub = CreateTube(0.45, 1.4, 16)
SetPosition(tub, 3.4, 0.8, 7)
SetEntityColor(tub, 120, 160, 180)

dsk = CreateDisk(0.8, 24)
SetPosition(dsk, 0, 0.02, 2)
SetEntityColor(dsk, 60, 80, 70)

While Not KeyDown(KEY_ESCAPE)
    dt# = DeltaTime() * 60
    TurnEntity(tor, 0.4 * dt, 0.8 * dt, 0)
    TurnEntity(sph, 0, 0.5 * dt, 0)
    RenderWorld
    Text(12, 12, "CreateBox Sphere Cylinder Cone Torus Capsule Pyramid Wedge Tube Disk")
    Text(12, 36, "Esc quit")
    Flip
Wend
End
