# Post-processing

OpenGL **3.3** fullscreen blit. After the G3N scene (and water FBO / shadows), the color buffer is sampled once.

![PostFX demo](images/postfx.png)

| Pass | What it is | What it is not |
| --- | --- | --- |
| Tonemap | Reinhard `c/(c+1)` after exposure | ACES filmic LUT |
| Bloom | 9-tap bright extract in the same shader | Separate half-res mip chain / compute |
| FXAA | Neighbor luma blend | SMAA / TAA |
| Grade | Contrast, saturation, RGB tint | Lift/gamma/gain wheels |

ImGui draws **after** the blit so editor panels stay unprocessed.

Commands: `EnablePostFX`, `SetExposure`, `SetBloom`, `SetFXAA`, `SetColorGrade`. See `docs/COMMANDS.md` for ranges and getters.

```powershell
.\bs.exe examples\postfx.bb
```

Arrow keys adjust exposure (up/down) and bloom (left/right). Close the window to quit.
