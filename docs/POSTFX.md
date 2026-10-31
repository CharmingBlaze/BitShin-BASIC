# Post-processing

OpenGL **3.3** fullscreen blit. After the G3N scene (and water FBO / shadows), the color buffer is sampled once. The scene target is half float (RGBA16F), as are the bloom buffers: lit shaders roll highlights off toward 2.0 instead of clipping at 1, so exposure and bloom grade the real brightness.

![PostFX demo](images/postfx.png)

| Pass | What it is | What it is not |
| --- | --- | --- |
| Tonemap | After exposure: `reinhard` (default, `c/(c+1)`), `neutral` (Khronos PBR Neutral), `aces` (fitted), `none` (clamp) | A separate filmic LUT texture |
| Bloom | Half-resolution chain (bright extract, then four smaller passes) added before tonemap. The 9-tap extract remains if the chain cannot be built | Separate compute bloom |
| FXAA | Neighbor luma blend | SMAA / TAA |
| Grade | Contrast, saturation, RGB tint | Lift/gamma/gain wheels |
| LUT | `SetLUT` reads a 3D colour table after the grade: built-in presets, `.cube` files, or PNG strips/tiles. `SetLUTAmount` blends it | A 1D curve or per-shot keyframes |
| Outline | `SetOutline` darkens pixels where linear depth jumps by more than a share of its distance, fading out toward the far plane | A normal-buffer or object-ID edge pass |
| Film | `SetFilmGrain`, `SetVignette`, `SetChromaticAberration`, `SetSharpen` after the grade | A deferred film graph |
| Contact | Half-resolution horizon occlusion (`EnableSSAO`) on the scene depth, blurred without crossing depth edges. It darkens fill light in creases | A deferred normal buffer |
| Exposure | `SetAutoExposure` eases the exposure toward a middle brightness from a center-weighted sample of the frame | A histogram or EV camera model |

ImGui draws **after** the blit so editor panels stay unprocessed.

Commands: `EnablePostFX`, `SetExposure`, `SetTonemap`, `SetBloom`, `SetFXAA`, `SetColorGrade`, `SetFilmGrain`, `SetVignette`, `SetChromaticAberration`, `SetSharpen`, `SetLUT`, `SetLUTAmount`, `ClearLUT`, `SetOutline`. Outdoor and indoor lighting presets store an exposure and the neutral tonemap; they do not enable this stack. See `docs/LIGHTING.md` and `docs/COMMANDS.md`.

```powershell
.\moon.exe examples\postfx.mb
```

Arrow keys adjust exposure (up/down) and bloom (left/right). Close the window to quit.
