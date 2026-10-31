package runtime

// Fixed texture units. G3N binds a material's own textures (MatTexture[],
// the PBR maps) from unit 0 up; everything here is bound by hand. Two
// samplers on one unit read the same texture: the FFT ocean once read the
// shadow map as wave height and rose 2 m. TestTextureUnitsDoNotClash checks
// programUnits against every program's GLSL.
const (
	// Lit (bsshadow), PBR (mbphysical), and terrain share the light units.
	unitShadow    = 8
	unitShadowDyn = 9
	unitSpec      = 10
	unitEmit      = 11
	unitCookie0   = 12
	unitCookie1   = 13
	unitNorm      = 14
	unitCaustic   = 15
	unitEnvCube   = 10 // PBR has no SpecMap

	// Terrain splat. G3N takes 0 and 1: EntityTexture, then a lightmap.
	unitTerrainSand   = 2
	unitTerrainGrass  = 3
	unitTerrainGrass2 = 4
	unitTerrainRock   = 5
	unitTerrainSnow   = 6
	unitTerrainRockN  = 7
	unitTerrainBlend  = 10

	// Water. Its material has no G3N textures, so 0-5 are free.
	unitWaterWake         = 5
	unitWaterRefract      = 6
	unitWaterReflect      = 7
	unitWaterDuDv         = 8
	unitWaterNormal       = 9
	unitWaterRefractDepth = 10
	unitWaterShadow       = 11
	unitWaterFFT          = 14

	unitAtmo = 5

	// GPU emitter sprites (raw GL program, no G3N textures).
	unitParticle = 1
)

// programUnits is each program's hand-bound samplers. g3nUnits is how many
// units from 0 G3N may fill with the material's own textures there.
var programUnits = map[string]struct {
	g3nUnits int
	samplers map[string]int
}{
	"bsshadow": {8, map[string]int{
		"ShadowMap": unitShadow, "ShadowMapDyn": unitShadowDyn, "SpecMap": unitSpec, "EmitMap": unitEmit,
		"SpotCookie0": unitCookie0, "SpotCookie1": unitCookie1, "NormMap": unitNorm, "WaterCaustic": unitCaustic,
	}},
	// Base color, metal-roughness, normal, emissive, occlusion, equirect env
	"mbphysical": {6, map[string]int{
		"ShadowMap": unitShadow, "ShadowMapDyn": unitShadowDyn, "uEnvCube": unitEnvCube,
		"SpotCookie0": unitCookie0, "SpotCookie1": unitCookie1, "WaterCaustic": unitCaustic,
	}},
	"mbterrain": {2, map[string]int{
		"TerrainSand": unitTerrainSand, "TerrainGrass": unitTerrainGrass, "TerrainGrass2": unitTerrainGrass2,
		"TerrainRock": unitTerrainRock, "TerrainSnow": unitTerrainSnow, "TerrainRockN": unitTerrainRockN,
		"TerrainBlendMap": unitTerrainBlend, "ShadowMap": unitShadow, "ShadowMapDyn": unitShadowDyn,
		"SpotCookie0": unitCookie0, "SpotCookie1": unitCookie1, "WaterCaustic": unitCaustic,
	}},
	"mbwater": {0, map[string]int{
		"WaterWake": unitWaterWake, "WaterRefract": unitWaterRefract, "WaterReflect": unitWaterReflect,
		"WaterDuDv": unitWaterDuDv, "WaterNormal": unitWaterNormal, "WaterRefractDepth": unitWaterRefractDepth,
		"ShadowMap": unitWaterShadow, "WaterFFT": unitWaterFFT, "SpotCookie0": unitCookie0, "SpotCookie1": unitCookie1,
	}},
	"mbatmo": {0, map[string]int{"AtmoT": unitAtmo}},
	"mbgpupart": {0, map[string]int{"PartTex": unitParticle}},
}

// g3nSamplers are bound by G3N from the material's own textures.
var g3nSamplers = map[string]bool{
	"MatTexture": true, "uBaseColorSampler": true, "uMetallicRoughnessSampler": true, "uNormalSampler": true,
	"uEmissiveSampler": true, "uOcclusionSampler": true, "uEnvSampler": true,
}
