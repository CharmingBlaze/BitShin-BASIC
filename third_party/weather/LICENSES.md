# Weather reference licenses

These repos were **not vendored**. Algorithms were reimplemented for GLSL 330 / G3N.

| Repo | License (upstream) | Use here |
| --- | --- | --- |
| ebruneton/precomputed_atmospheric_scattering | BSD-3-Clause (Eric Bruneton, 2017) | Transmittance idea + single scatter; no 4D tables copied |
| kevinmoran/Clouds, RainShader, Fog, Wind | Sample code (reimplemented) | Layered clouds, streaks/wetness, height fog, sway |
| Polytonic/ParticleSystem | Typical MIT sample | Mapped to existing emitters |
| fogleman/lightning | Midpoint-displacement algorithm | Bolt polylines + flash |

See `docs/WEATHER.md`.
