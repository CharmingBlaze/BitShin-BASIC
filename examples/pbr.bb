; PBR — metallic-roughness spheres on mbphysical with sun + CSM.
; Leave running (no Escape) — close the window to quit.

Graphics3D(960, 600)
SetWindowTitle("BitShin BASIC — PBR")
SetCameraClsColor(12, 14, 22)
SetAmbientLight(22, 26, 34)

; Camera
cam = CreateCamera().Position([0, 3.2, -8])
cam.Point(0, 0.6, 3)

; Shadows / IBL
EnableShadows True
ShadowCascades 2
ShadowMapSize 1024
SetShadowBias 0.003
SetShadowPCF 3

sun = CreateDirectionalLight()
SetLightDirection sun, 50, 35, 0
SetLightColor sun, 255, 230, 190
SetLightShadow sun, True

CreateSkyBox("default")
SetIBL True
SetIBLIntensity 1.0

; Ground
ground = CreatePlane().Scale(18, 1, 18).Position([0, 0, 4])
SetMaterialPBR(ground, 1)
SetAlbedo(ground, 48, 54, 64)
SetMetallic(ground, 0)
SetRoughness(ground, 0.88)
SetAO(ground, 1)

; Rough dielectric
clay = CreateSphere().Position([-3.2, 0.75, 4])
SetMaterialPBR(clay, 1)
SetAlbedo(clay, 200, 72, 58)
SetMetallic(clay, 0)
SetRoughness(clay, 0.86)

; Smooth metal
gold = CreateSphere().Position([-1.05, 0.75, 4])
SetAlbedo(gold, 232, 196, 96)
SetMetallic(gold, 1)
SetRoughness(gold, 0.12)

; Rusted metal (library material + SetMaterial)
rustMat = CreatePBRMaterial()
SetBaseColor(rustMat, 138, 74, 42)
SetMetallic(rustMat, 0.62)
SetRoughness(rustMat, 0.74)
SetAO(rustMat, 0.55)
rust = CreateSphere().Position([1.1, 0.75, 4])
SetMaterial(rust, rustMat)

; Emissive
glow = CreateSphere().Position([3.25, 0.75, 4])
SetAlbedo(glow, 16, 18, 24)
SetMetallic(glow, 0)
SetRoughness(glow, 0.35)
SetEmissive(glow, 40, 160, 255)

Print "PBR: clay / gold / rust / emissive  |  GetMaterialPBR(gold)=" + Str(GetMaterialPBR(gold))

; Loop
While 1
    dt# = DeltaTime() * 60
    clay.Turn(0, 0.35 * dt, 0)
    gold.Turn(0, 0.35 * dt, 0)
    rust.Turn(0, 0.35 * dt, 0)
    glow.Turn(0, 0.35 * dt, 0)
    UpdateWorld
    RenderWorld
    Flip
Wend
