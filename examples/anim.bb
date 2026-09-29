; Animation — load hero.glb / hero.gltf with clips, or a spinning cube fallback.
; Esc quits.

Graphics3D(800, 600)
SetBuffer(BackBuffer())
SetWindowTitle("BitShin BASIC — animation")

; Camera / light
camera = CreateCamera().Position([0, 1.4, -4])
light = CreateLight().Rotate(50, 30, 0)

; Ground
ground = CreatePlane(12, 12).Position([0, 0, 4]).Color(36, 42, 52)

; Mesh — prefer glTF/GLB with animation clips
If FileExists("hero.glb") = 1 Then
    hero = LoadAnimation("hero.glb")
    PlayAnimation(hero, 1, 1)
ElseIf FileExists("hero.gltf") = 1 Then
    hero = LoadAnimation("hero.gltf")
    PlayAnimation(hero, 1, 1)
Else
    hero = CreateCube().Position([0, 1, 4]).Color(80, 180, 255)
EndIf

; Loop
While Not KeyDown(KEY_ESCAPE)
    dt# = DeltaTime() * 60
    If FileExists("hero.glb") = 0 And FileExists("hero.gltf") = 0 Then
        hero.Turn(0, 0.6 * dt, 0)
    EndIf
    UpdateWorld
    RenderWorld
    Text(12, 12, "AnimTime=" + GetAnimationTime(hero) + "  length=" + GetAnimationLength(hero))
    Text(12, 28, "Supply hero.glb / hero.gltf with clips for LoadAnimation")
    Text(12, 44, "SetAnimBlend e, w [, seq] nlerps the previous clip into the current one")
    Flip
Wend
End
