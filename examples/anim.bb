; Mesh animation — put a glTF/GLB with clips next to this file as hero.glb
; LoadAnimation / PlayAnimation / GetAnimationTime. Esc quits.

Graphics3D(800, 600)
SetBuffer(BackBuffer())
SetWindowTitle("BitShin BASIC — animation")

camera = CreateCamera()
SetPosition(camera, 0, 1.4, -4)
light = CreateLight()
SetRotation(light, 50, 30, 0)

ground = CreatePlane(12, 12)
SetPosition(ground, 0, 0, 4)
SetEntityColor(ground, 36, 42, 52)

If FileExists("hero.glb") = 1 Then
    hero = LoadAnimation("hero.glb")
    PlayAnimation(hero, 1, 1)
ElseIf FileExists("hero.gltf") = 1 Then
    hero = LoadAnimation("hero.gltf")
    PlayAnimation(hero, 1, 1)
Else
    hero = CreateCube()
    SetPosition(hero, 0, 1, 4)
    SetEntityColor(hero, 80, 180, 255)
EndIf

While Not KeyDown(KEY_ESCAPE)
    dt# = DeltaTime() * 60
    If FileExists("hero.glb") = 0 And FileExists("hero.gltf") = 0 Then
        TurnEntity(hero, 0, 0.6 * dt, 0)
    EndIf
    UpdateWorld
    RenderWorld
    Text(12, 12, "AnimTime=" + GetAnimationTime(hero) + "  length=" + GetAnimationLength(hero))
    Text(12, 28, "Supply hero.glb / hero.gltf with clips for LoadAnimation")
    Flip
Wend
End
