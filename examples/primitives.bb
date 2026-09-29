; Primitives — mesh showcase (box, sphere, cylinder, cone, torus, and more).
; Esc quits.

Graphics3D(960, 600)
SetBuffer(BackBuffer())
SetWindowTitle("BitShin BASIC — primitives")
SetCameraClsColor(18, 22, 32)
SetAmbientLight(50, 55, 70)

; Camera / sky / fog
cam = CreateCamera().Position([0, 3, -12])
CreateSkyBox()
CameraFogMode(cam, 1)
CameraFogColor(cam, 18, 22, 32)
CameraFogRange(cam, 14, 40)

; Light
sun = CreateDirectionalLight()
SetLightDirection(sun, 40, 30, 0)
SetLightColor(sun, 255, 240, 210)

; World
ground = CreatePlane(28, 28).Position([0, 0, 5]).Color(32, 38, 48)

box = CreateBox(1.4, 0.8, 1.1).Position([-4, 0.6, 4]).Color(220, 90, 80)
sph = CreateSphere(1, 16, 0).Position([-2, 0.8, 4]).Color(80, 180, 220)
cyl = CreateCylinder(0.45, 1.6, 14).Position([0, 0.8, 4]).Color(90, 200, 120)
cone = CreateCone(0.55, 1.6, 12).Position([2, 0.8, 4]).Color(230, 190, 70)
tor = CreateTorus(0.7, 0.22, 14, 20).Position([4, 0.9, 4]).Color(200, 120, 220)
cap = CreateCapsule(0.35, 0.9, 12).Position([-3, 0.9, 7]).Color(240, 240, 240)
pyr = CreatePyramid(1.4).Position([-1, 0.7, 7]).Color(70, 140, 200)
wed = CreateWedge(1.4, 0.8, 1.2).Position([1.4, 0.4, 7]).Color(180, 100, 70)
tub = CreateTube(0.45, 1.4, 16).Position([3.4, 0.8, 7]).Color(120, 160, 180)
dsk = CreateDisk(0.8, 24).Position([0, 0.02, 2]).Color(60, 80, 70)

; Loop
While Not KeyDown(KEY_ESCAPE)
    dt# = DeltaTime() * 60
    tor.Turn(0.4 * dt, 0.8 * dt, 0)
    sph.Turn(0, 0.5 * dt, 0)
    RenderWorld
    Text(12, 12, "CreateBox Sphere Cylinder Cone Torus Capsule Pyramid Wedge Tube Disk")
    Text(12, 36, "Esc quit")
    Flip
Wend
End
