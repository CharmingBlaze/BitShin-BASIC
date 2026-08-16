; Shader programs (not an Unreal graph): CreateShader + SetShader + SetShaderUniform.

Graphics3D(800, 600)
SetWindowTitle("BitShin BASIC — Shaders")
cam = CreateCamera()
SetPosition(cam, 0, 1.4, -5)
light = CreateLight()
SetRotation(light, 40, 20, 0)

ball = CreateSphere()
SetPosition(ball, 0, 0, 4)
sh = CreateShader()
SetShader(ball, sh)
SetShaderUniform(sh, "LightDir", 0.2, 0.9, 0.3)

While Not WindowShouldClose()
    TurnEntity(ball, 0.3, 0.5, 0)
    RenderWorld
    Text(12, 12, "CreateShader / SetShader  ok=" + Str(ShaderOK(sh)))
    Flip
Wend
End
