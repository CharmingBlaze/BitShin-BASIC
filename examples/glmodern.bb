; Modern GL demo. Runs on OpenGL 3.3. Compute/SSBO/tess return 0 on old GPUs.

Graphics3D(960, 600)
SetWindowTitle("Modern GL")
SetCameraClsColor(40, 48, 62)
SetAmbientLight(50, 55, 70)

cam = CreateCamera()
light = CreateDirectionalLight()
SetLightDirection(light, 40, 25, 0)

Print("GL " + GLVersion$())
Print("renderer " + GLRenderer$())
Print("compute " + Str(GLHasCompute()) + " ssbo " + Str(GLHasSSBO()) + " ubo " + Str(GLHasUBO()))
Print("instance " + Str(GLHasInstancing()) + " geom " + Str(GLHasGeometry()) + " tess " + Str(GLHasTessellation()))

src$ = "#version 330 core" + Chr(10) + "void main() { gl_Position = vec4(0.0); }" + Chr(10)
sh = CompileShader(src$, "vert")
Print("CompileShader " + Str(sh) + " " + ShaderLog$(sh) + " spirv " + Str(ShaderSPIRVSize(sh)))

ubo = CreateUniformBuffer(8)
SetUniformBuffer(ubo, 0, 1, 2, 3, 4)
BindUniformBuffer(ubo, 0)

ssbo = CreateStorageBuffer(64)
SetStorageBuffer(ssbo, 0, 0.5)
BindStorageBuffer(ssbo, 0)
cs = CreateComputeShader("")
If cs <> 0 Then
    DispatchCompute(cs, 1, 1, 1)
    Print("compute ok " + Str(GetStorageBuffer(ssbo, 0)))
Else
    Print("compute skipped (3.3 is fine)")
EndIf

EnableTessellation(True)
Print("tess " + Str(Tessellation()))

proto = CreateCube()
HideEntity(proto)
batch = CreateInstancedMesh(proto, 12)
For i = 0 To 11
    SetInstanceData(batch, i, (i Mod 4) * 1.6 - 2.4, 0, Floor(i / 4) * 1.6 + 4, 0, i * 15, 0, 0.4, 0.4, 0.4)
Next
BatchInstances(batch)
EnableGPUInstances(True)

pts = CreateGeomPoints(8)
For i = 0 To 7
    SetGeomPoint(pts, i, Sin(i * 45) * 3, 1.2, Cos(i * 45) * 3 + 5, 0.2, 255, 180, 60)
Next

Print("NoiseFBM " + Str(NoiseFBM(2, 3, 4)))

yaw# = 0
While Not WindowShouldClose()
    dt# = DeltaTime()
    yaw = yaw + 25 * dt
    TurnEntity(batch, 0, 12 * dt, 0)
    CameraFollow(cam, batch, 10, 4, 8, yaw, 12)
    GuiBegin("GL")
    GuiText(GLVersion$())
    GuiText("compute " + Str(GLHasCompute()) + "  gpu inst " + Str(GPUInstances()))
    GuiEnd()
    RenderWorld
    Flip
Wend
End
