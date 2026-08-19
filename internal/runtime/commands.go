package runtime

import (
	"math"
	"time"

	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/texture"
	"github.com/g3n/engine/window"

	"bitshinbasic/internal/value"
)

func (w *World) commandTable() map[string]cmd {
	n := func(fn func([]value.Value) (value.Value, error)) cmd { return fn }
	z := func() (value.Value, error) { return value.Num(0), nil }
	need := func(fn func([]value.Value) (value.Value, error)) cmd {
		return func(a []value.Value) (value.Value, error) {
			if err := w.require(); err != nil {
				return value.Value{}, err
			}
			return fn(a)
		}
	}

	m := map[string]cmd{
		"graphics3d": n(func(a []value.Value) (value.Value, error) {
			return w.graphics3D(argI(a, 0, 800), argI(a, 1, 600), argI(a, 2, 0), argI(a, 3, 2))
		}),
		"apptitle": n(func(a []value.Value) (value.Value, error) {
			w.title = argS(a, 0)
			if w.app != nil {
				if gw, ok := w.app.IWindow.(*window.GlfwWindow); ok {
					gw.SetTitle(w.title)
				}
			}
			return z()
		}),
		"endgraphics": n(func(a []value.Value) (value.Value, error) {
			w.quit = true
			return z()
		}),
		"createcamera": need(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.spawnCamera(argI(a, 0, 0)))), nil
		}),
		"createfreecamera": need(func(a []value.Value) (value.Value, error) {
			id := w.spawnCamera(argI(a, 0, 0))
			w.armFreeLook()
			return value.Num(float64(id)), nil
		}),
		"updatefreelook": need(func(a []value.Value) (value.Value, error) {
			return w.updateFreeLook(a)
		}),
		"createlight": need(func(a []value.Value) (value.Value, error) {
			kind := argI(a, 0, 1)
			if kind < 1 || kind > 3 {
				kind = 1
			}
			return value.Num(float64(w.makeLight(kind, argI(a, 1, 0)))), nil
		}),
		"createcube":     need(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.createCubeMesh(a))), nil }),
		"createbox":      need(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.createBoxMesh(a))), nil }),
		"createsphere":   need(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.createSphereMesh(a))), nil }),
		"createcylinder": need(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.createCylinderMesh(a))), nil }),
		"createcone":     need(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.createConeMesh(a))), nil }),
		"createplane":    need(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.createPlaneMesh(a))), nil }),
		"createtorus":    need(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.createTorusMesh(a))), nil }),
		"createcapsule":  need(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.createCapsuleMesh(a))), nil }),
		"createdisk":     need(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.createDiskMesh(a))), nil }),
		"createcircle":   need(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.createDiskMesh(a))), nil }),
		"createpyramid":  need(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.createPyramidMesh(a))), nil }),
		"createwedge":    need(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.createWedgeMesh(a))), nil }),
		"createtube":     need(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.createTubeMesh(a))), nil }),
		"createquad":     need(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.createQuadMesh(a))), nil }),
		"createpivot": need(func(a []value.Value) (value.Value, error) {
			id := w.addEntity(&Entity{node: core.NewNode()}, argI(a, 0, 0))
			return value.Num(float64(id)), nil
		}),
		"loadmesh": need(func(a []value.Value) (value.Value, error) {
			id, err := w.loadMeshFile(argS(a, 0), argI(a, 1, 0))
			return value.Num(float64(id)), err
		}),
		"copyentity": need(func(a []value.Value) (value.Value, error) {
			src, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if src.mesh == nil {
				id := w.addEntity(&Entity{node: core.NewNode()}, argI(a, 1, 0))
				return value.Num(float64(id)), nil
			}
			if src.usePBR && src.pbrWrap != nil {
				pm := w.newPBR()
				pm.metallic = src.pbrWrap.metallic
				pm.roughness = src.pbrWrap.roughness
				pm.ao = src.pbrWrap.ao
				pm.albedo = src.pbrWrap.albedo
				pm.emissive = src.pbrWrap.emissive
				pm.ibl = src.pbrWrap.ibl
				pm.applyFactors()
				mesh := graphic.NewMesh(src.mesh.GetGeometry(), pm)
				id := w.addEntity(&Entity{node: mesh, mesh: mesh, pbr: pm.Physical, pbrWrap: pm, usePBR: true}, argI(a, 1, 0))
				return value.Num(float64(id)), nil
			}
			mat := w.newMat()
			if src.mat != nil {
				c := src.mat.AmbientColor()
				mat.SetColor(&c)
			}
			mesh := graphic.NewMesh(src.mesh.GetGeometry(), newLitMat(w, mat))
			id := w.addEntity(&Entity{node: mesh, mesh: mesh, mat: mat}, argI(a, 1, 0))
			return value.Num(float64(id)), nil
		}),
		"freeentity": need(func(a []value.Value) (value.Value, error) {
			w.freeEntityID(argI(a, 0, 0))
			return z()
		}),
		"hideentity": need(func(a []value.Value) (value.Value, error) {
			n, err := w.nodeOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			n.SetVisible(false)
			return z()
		}),
		"showentity": need(func(a []value.Value) (value.Value, error) {
			n, err := w.nodeOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			n.SetVisible(true)
			return z()
		}),
		"positionentity": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			gx, gy, gz := toG3N(float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)))
			e.node.GetNode().SetPosition(gx, gy, gz)
			if e.parent == 0 && w.phys3 != nil {
				if _, _, _, ok := w.phys3.GetPosition(argI(a, 0, 0)); ok {
					w.phys3.SetPosition(argI(a, 0, 0), float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)))
					w.phys3.Wake(argI(a, 0, 0))
				}
			}
			w.aimDefaultCamera(e)
			return z()
		}),
		"moveentity": need(func(a []value.Value) (value.Value, error) {
			n, err := w.nodeOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			n.TranslateX(float32(argN(a, 1, 0)))
			n.TranslateY(float32(argN(a, 2, 0)))
			n.TranslateZ(-float32(argN(a, 3, 0)))
			return z()
		}),
		"translateentity": need(func(a []value.Value) (value.Value, error) {
			n, err := w.nodeOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			p := n.Position()
			gx, gy, gz := toG3N(float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)))
			n.SetPosition(p.X+gx, p.Y+gy, p.Z+gz)
			return z()
		}),
		"turnentity": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			e.pitch += float32(argN(a, 1, 0))
			e.yaw += float32(argN(a, 2, 0))
			e.roll += float32(argN(a, 3, 0))
			w.applyRot(e)
			return z()
		}),
		"rotateentity": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			e.pitch = float32(argN(a, 1, 0))
			e.yaw = float32(argN(a, 2, 0))
			e.roll = float32(argN(a, 3, 0))
			w.applyRot(e)
			return z()
		}),
		"scaleentity": need(func(a []value.Value) (value.Value, error) {
			n, err := w.nodeOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			n.SetScale(float32(argN(a, 1, 1)), float32(argN(a, 2, 1)), float32(argN(a, 3, 1)))
			return z()
		}),
		"pointentity": need(func(a []value.Value) (value.Value, error) {
			return w.pointOrLook(a)
		}),
		"camerafollow": need(func(a []value.Value) (value.Value, error) {
			return w.cameraFollow(a)
		}),
		"entitycolor": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			w.setEntityRGB(e, float32(argN(a, 1, 255)), float32(argN(a, 2, 255)), float32(argN(a, 3, 255)))
			return z()
		}),
		"entityalpha": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			a0 := float32(argN(a, 1, 1))
			e.tint.A = a0
			if e.mat != nil {
				e.mat.SetOpacity(a0)
				e.mat.SetTransparent(a0 < 1)
			}
			if e.pbr != nil {
				e.pbr.SetBaseColorFactor(&e.tint)
				e.pbr.SetTransparent(a0 < 1)
			}
			return z()
		}),
		"entityx": need(func(a []value.Value) (value.Value, error) {
			n, err := w.nodeOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			p := n.Position()
			if argI(a, 1, 0) != 0 {
				w.refreshWorldMatrices()
				p = worldPos(n)
			}
			x, _, _ := fromG3N(p.X, p.Y, p.Z)
			return value.Num(float64(x)), nil
		}),
		"entityy": need(func(a []value.Value) (value.Value, error) {
			n, err := w.nodeOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			p := n.Position()
			if argI(a, 1, 0) != 0 {
				w.refreshWorldMatrices()
				p = worldPos(n)
			}
			return value.Num(float64(p.Y)), nil
		}),
		"entityz": need(func(a []value.Value) (value.Value, error) {
			n, err := w.nodeOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			p := n.Position()
			if argI(a, 1, 0) != 0 {
				w.refreshWorldMatrices()
				p = worldPos(n)
			}
			_, _, z := fromG3N(p.X, p.Y, p.Z)
			return value.Num(float64(z)), nil
		}),
		"entitypitch": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(e.pitch)), nil
		}),
		"entityyaw": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(e.yaw)), nil
		}),
		"entityroll": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(e.roll)), nil
		}),
		"entitydistance": need(func(a []value.Value) (value.Value, error) {
			na, err := w.nodeOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			nb, err := w.nodeOf(argI(a, 1, 0))
			if err != nil {
				return value.Value{}, err
			}
			pa, pb := worldPos(na), worldPos(nb)
			dx, dy, dz := pa.X-pb.X, pa.Y-pb.Y, pa.Z-pb.Z
			return value.Num(math.Sqrt(float64(dx*dx + dy*dy + dz*dz))), nil
		}),
		"cameraclscolor": need(func(a []value.Value) (value.Value, error) {
			off := 0
			if len(a) >= 4 {
				off = 1
			}
			c := rgb(argN(a, off, 0), argN(a, off+1, 0), argN(a, off+2, 0))
			w.clear = *c
			return z()
		}),
		"ambientlight": need(func(a []value.Value) (value.Value, error) {
			c := rgb(argN(a, 0, 255), argN(a, 1, 255), argN(a, 2, 255))
			if w.ambient != nil {
				w.ambient.SetColor(c)
				w.ambient.SetIntensity(1)
			}
			w.wx.ambSaved = false
			return z()
		}),
		"setambientcolor": need(func(a []value.Value) (value.Value, error) {
			c := rgb(argN(a, 0, 0.24), argN(a, 1, 0.28), argN(a, 2, 0.36))
			if w.ambient != nil {
				w.ambient.SetColor(c)
				w.ambient.SetIntensity(1)
			}
			w.wx.ambSaved = false
			return z()
		}),
		"wireframe": need(func(a []value.Value) (value.Value, error) {
			w.wire = argI(a, 0, 1) != 0
			for _, e := range w.ents {
				if e.mat != nil {
					e.mat.SetWireframe(w.wire)
				}
				if e.pbr != nil {
					e.pbr.SetWireframe(w.wire)
				}
			}
			return z()
		}),
		"loadtexture": need(func(a []value.Value) (value.Value, error) {
			file := argS(a, 0)
			if len(a) >= 2 && a[0].Kind != value.KindStr {
				file = argS(a, 1)
			}
			path, err := w.openPath(file)
			if err != nil {
				return value.Num(0), nil
			}
			tex, err := texture.NewTexture2DFromImage(path)
			if err != nil {
				return value.Num(0), nil
			}
			id := w.nextTex
			w.nextTex++
			w.texs[id] = &texSlot{tex: tex, path: file}
			return value.Num(float64(id)), nil
		}),
		"entitytexture": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			t := w.texs[argI(a, 1, 0)]
			if t != nil && e.mat != nil {
				e.mat.AddTexture(t.tex)
				e.albedoTex = t.tex
			}
			if t != nil && e.pbr != nil {
				e.pbr.SetBaseColorMap(t.tex)
			}
			return z()
		}),
		"entitytype": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			e.etype = argI(a, 1, 0)
			return z()
		}),
		"entityradius": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			e.radius = float32(argN(a, 1, 1))
			return z()
		}),
		"collisions": n(func(a []value.Value) (value.Value, error) {
			w.rules = append(w.rules, collideRule{
				src: argI(a, 0, 0), dest: argI(a, 1, 0),
				method: argI(a, 2, 1), response: argI(a, 3, 1),
			})
			return z()
		}),
		"countcollisions": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(len(e.collided))), nil
		}),
		"entitycollided": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if len(e.collided) == 0 {
				return value.Num(0), nil
			}
			if len(a) < 2 {
				return value.Num(float64(e.collided[0])), nil
			}
			filter := argI(a, 1, 0)
			for i := 0; i < len(e.collided); i++ {
				id := e.collided[i]
				if id == filter {
					return value.Num(float64(id)), nil
				}
				if other := w.ents[id]; other != nil && other.etype == filter {
					return value.Num(float64(id)), nil
				}
			}
			return value.Num(0), nil
		}),
		"updateworld": n(func(a []value.Value) (value.Value, error) {
			w.updateWorld()
			return z()
		}),
		"renderworld": n(func(a []value.Value) (value.Value, error) { return z() }),
		"flip": n(func(a []value.Value) (value.Value, error) {
			w.markFlip()
			return z()
		}),
		"keydown": n(func(a []value.Value) (value.Value, error) {
			if w.keyDown(argI(a, 0, 0)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"keyhit": n(func(a []value.Value) (value.Value, error) {
			if w.keyHit(argI(a, 0, 0)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"mousedown": n(func(a []value.Value) (value.Value, error) {
			if w.guiCapturesMouse() {
				return value.Num(0), nil
			}
			b := argI(a, 0, 1)
			if b >= 0 && b < len(w.mouse) && w.mouse[b] {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"mousex": n(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.mx)), nil }),
		"mousey": n(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.my)), nil }),
		"mousez": n(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.mz)), nil }),
		"graphicswidth": n(func(a []value.Value) (value.Value, error) {
			if w.app != nil {
				ww, _ := w.app.GetSize()
				return value.Num(float64(ww)), nil
			}
			return value.Num(float64(w.scrW)), nil
		}),
		"graphicsheight": n(func(a []value.Value) (value.Value, error) {
			if w.app != nil {
				_, hh := w.app.GetSize()
				return value.Num(float64(hh)), nil
			}
			return value.Num(float64(w.scrH)), nil
		}),
		"deltatime": n(func(a []value.Value) (value.Value, error) {
			if w.delta <= 0 {
				return value.Num(1.0 / 60.0), nil
			}
			return value.Num(w.delta), nil
		}),
		"frametime": n(func(a []value.Value) (value.Value, error) {
			if w.delta <= 0 {
				return value.Num(1.0 / 60.0), nil
			}
			return value.Num(w.delta), nil
		}),
		"millisecs": n(func(a []value.Value) (value.Value, error) {
			if w.started.IsZero() {
				w.started = time.Now()
			}
			return value.Num(float64(time.Since(w.started).Milliseconds())), nil
		}),
		"color": n(func(a []value.Value) (value.Value, error) {
			r, g, b := argN(a, 0, 255), argN(a, 1, 255), argN(a, 2, 255)
			w.textRGB = *rgb(r, g, b)
			w.drawRGB = rgbBytes(r, g, b)
			return z()
		}),
		"text": n(func(a []value.Value) (value.Value, error) {
			if w.mode2D {
				w.draws = append(w.draws, drawOp{kind: 4, x: float32(argN(a, 0, 0)), y: float32(argN(a, 1, 0)), text: argS(a, 2)})
				return z()
			}
			if err := w.require3D(); err != nil {
				return value.Value{}, err
			}
			lab := gui.NewLabel(argS(a, 2))
			lab.SetPosition(float32(argN(a, 0, 0)), float32(argN(a, 1, 0)))
			lab.SetColor(&w.textRGB)
			w.scene.Add(lab)
			w.texts = append(w.texts, lab)
			return z()
		}),
		"hudprint": n(func(a []value.Value) (value.Value, error) {
			w.hudPrint(argS(a, 0))
			return z()
		}),
		"cls": n(func(a []value.Value) (value.Value, error) {
			w.draws = w.draws[:0]
			w.clearFrameText()
			w.clearHudPrint()
			return z()
		}),
	}
	for k, v := range w.classic3DCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.physCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.vehicleCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.ropeCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.clothCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.ezHelperCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.characterCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.netCommands(n, z) {
		m[k] = v
	}
	for k, v := range w.twoDCommands(n, z) {
		m[k] = v
	}
	for k, v := range w.timeCommands(n, z) {
		m[k] = v
	}
	for k, v := range w.audioCommands(n, z) {
		m[k] = v
	}
	for k, v := range w.animCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.fxCommands(n, z, need) {
		m[k] = v
	}
	for _, pair := range [][2]string{
		{"particle2drate", "emitterrate"},
		{"particle2dmax", "emittermax"},
		{"particle2dlife", "emitterlife"},
		{"particle2dsize", "emittersize"},
		{"particle2dcolor", "emittercolor"},
		{"particle2dvelocity", "emittervelocity"},
		{"particle2dgravity", "emittergravity"},
		{"particle2dburst", "emitterburst"},
		{"particle2dsprite", "emitterparticle"},
		{"positionemitter2d", "positionemitter"},
		{"freeemitter2d", "freeemitter"},
	} {
		if m[pair[0]] == nil && m[pair[1]] != nil {
			m[pair[0]] = m[pair[1]]
		}
	}
	for k, v := range w.tileCommands(n, z) {
		m[k] = v
	}
	for k, v := range w.xformCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.navCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.agentCommands(n, z) {
		m[k] = v
	}
	for k, v := range w.dataCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.poolCommands(n, z) {
		m[k] = v
	}
	for k, v := range w.windowCommands(n, z) {
		m[k] = v
	}
	for k, v := range w.extraWindowCommands(n, z) {
		m[k] = v
	}
	for k, v := range w.guiCommands(n, z) {
		m[k] = v
	}
	for k, v := range w.g3nGuiCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.inputCommands(n, z) {
		m[k] = v
	}
	for k, v := range w.ecsCommands(n, z) {
		m[k] = v
	}
	for k, v := range w.lightCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.pbrCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.jobCommands(n, z) {
		m[k] = v
	}
	for k, v := range w.streamCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.instanceCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.probeCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.terrainCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.heightmapCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.geoCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.procTexCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.waterCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.crowdCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.scaleCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.glmodernCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.cloudCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.postCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.shaderCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.cameraCommands(n, z, need) {
		m[k] = v
	}
	for k, v := range w.gameplayCommands(n, z, need) {
		m[k] = v
	}
	applyModernAliases(m)
	return m
}

func (w *World) spawnCamera(parent int) int {
	aspect := float32(16) / 9
	if w.app != nil {
		ww, hh := w.app.GetSize()
		if hh > 0 {
			aspect = float32(ww) / float32(hh)
		}
	}
	cam := camera.New(aspect)
	cam.SetFar(4000)
	id := w.addEntity(&Entity{node: cam, cam: cam, kind: "camera"}, parent)
	if w.cam == nil {
		w.cam = cam
	}
	if w.listenEnt == 0 {
		w.listenEnt = id
	}
	w.aimDefaultCamera(w.ents[id])
	return id
}

// aimDefaultCamera points an unrotated Blitz camera along +Z.
// Only the showcase pose (about 0,2,-6) looks at the origin so a cube
// there is centered. High chase cameras (car/boat/tank) look forward
// at ground height — looking at the origin from (0,8,-16) puts the
// vehicle above the ray and the playfield out of frame.
// PointEntity / SetRotation still win afterwards (claw.bb).
func (w *World) aimDefaultCamera(e *Entity) {
	if e == nil || e.cam == nil {
		return
	}
	if e.pitch != 0 || e.yaw != 0 || e.roll != 0 {
		return
	}
	n := e.node.GetNode()
	p := n.Position()
	bx, by, bz := fromG3N(p.X, p.Y, p.Z)
	lookY, lookZ := by, bz+8
	if by >= 1.2 && by <= 3.6 && bz <= -4 && bz >= -9 {
		lookY = 0
		lookZ = 0
	} else if by >= 3 {
		lookY = 1
		lookZ = bz + 24
	}
	tx, ty, tz := toG3N(bx, lookY, lookZ)
	up := math32.Vector3{0, 1, 0}
	n.LookAt(&math32.Vector3{tx, ty, tz}, &up)
}

func (w *World) hudPrint(line string) {
	w.hudLines = append(w.hudLines, line)
}

func (w *World) clearHudPrint() {
	w.dropHudLabs()
	w.hudLines = w.hudLines[:0]
}

func (w *World) dropHudLabs() {
	for _, t := range w.hudLabs {
		if t == nil {
			continue
		}
		if p := t.Parent(); p != nil {
			p.GetNode().Remove(t)
		}
		t.SetVisible(false)
	}
	w.hudLabs = w.hudLabs[:0]
}

// flushHudPrint rebuilds Print() labels when the GL context is current
// (inside Flip/render). Creating them at Graphics3D/Print time is silent.
func (w *World) flushHudPrint() {
	if w.mode2D || w.scene == nil {
		return
	}
	w.dropHudLabs()
	for i := 0; i < len(w.hudLines); i++ {
		w.placeHudLabel(w.hudLines[i], i)
	}
}

func (w *World) placeHudLabel(line string, row int) {
	if w.scene == nil {
		return
	}
	lab := gui.NewLabel(line)
	lab.SetPosition(0, float32(row)*16)
	lab.SetColor(&w.textRGB)
	w.scene.Add(lab)
	w.hudLabs = append(w.hudLabs, lab)
}

// CameraFollow cam, target, dist, height [, damp, yaw, pitch]
// Lakitu-style: orbit yaw/pitch set the follow point; the camera eases there.
func (w *World) cameraFollow(a []value.Value) (value.Value, error) {
	camE, err := w.ent(argI(a, 0, 0))
	if err != nil {
		return value.Value{}, err
	}
	tgt, err := w.nodeOf(argI(a, 1, 0))
	if err != nil {
		return value.Value{}, err
	}
	dist := argN(a, 2, 8)
	if dist < 1.2 {
		dist = 1.2
	}
	height := argN(a, 3, 3.1)
	damp := argN(a, 4, 9)
	yaw := argN(a, 5, float64(camE.yaw))
	pitch := argN(a, 6, 14)
	if pitch > 52 {
		pitch = 52
	}
	if pitch < -22 {
		pitch = -22
	}
	tp := worldPos(tgt)
	tx, ty, tz := fromG3N(tp.X, tp.Y, tp.Z)
	pr := pitch * math.Pi / 180
	yr := yaw * math.Pi / 180
	horiz := dist * math.Cos(pr)
	wishX := float64(tx) - math.Sin(yr)*horiz
	wishY := float64(ty) + height + dist*math.Sin(pr)*0.35
	wishZ := float64(tz) - math.Cos(yr)*horiz
	minY := float64(ty) + 0.65
	if wishY < minY {
		wishY = minY
	}
	cn := camE.node.GetNode()
	cp := worldPos(cn)
	cx, cy, cz := fromG3N(cp.X, cp.Y, cp.Z)
	dt := w.delta
	if dt <= 0 {
		dt = 0.016
	}
	if dt > 0.05 {
		dt = 0.05
	}
	k := 1.0
	if camE.camFollowed && damp > 0 {
		k = 1 - math.Exp(-damp*dt)
	}
	camE.camFollowed = true
	nx := float64(cx) + (wishX-float64(cx))*k
	ny := float64(cy) + (wishY-float64(cy))*k
	nz := float64(cz) + (wishZ-float64(cz))*k
	if ny < minY {
		ny = minY
	}
	gx, gy, gz := toG3N(float32(nx), float32(ny), float32(nz))
	cn.SetPosition(gx, gy, gz)
	lookX, lookY, lookZ := toG3N(tx, ty+1.15, tz)
	up := math32.Vector3{0, 1, 0}
	cn.LookAt(&math32.Vector3{lookX, lookY, lookZ}, &up)
	camE.yaw = float32(yaw)
	camE.pitch = float32(pitch)
	return value.Num(0), nil
}

func (w *World) pointOrLook(a []value.Value) (value.Value, error) {
	n, err := w.nodeOf(argI(a, 0, 0))
	if err != nil {
		return value.Value{}, err
	}
	var p math32.Vector3
	if len(a) >= 4 {
		gx, gy, gz := toG3N(float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)))
		p = math32.Vector3{gx, gy, gz}
	} else {
		tgt, err := w.nodeOf(argI(a, 1, 0))
		if err != nil {
			return value.Value{}, err
		}
		p = worldPos(tgt)
	}
	up := math32.Vector3{0, 1, 0}
	n.LookAt(&p, &up)
	return value.Num(0), nil
}

func (w *World) setEntityRGB(e *Entity, r, g, b float32) {
	if e == nil {
		return
	}
	c := rgb(float64(r), float64(g), float64(b))
	e.tint.R, e.tint.G, e.tint.B = c.R, c.G, c.B
	if e.mat != nil {
		e.mat.SetColor(c)
		e.mat.SetEmissiveColor(&math32.Color{0, 0, 0})
	}
	if e.pbr != nil {
		e.pbr.SetBaseColorFactor(&e.tint)
	}
}

func (w *World) applyRot(e *Entity) {
	n := e.node.GetNode()
	if e.cam != nil {
		w.aimCameraEuler(e)
		return
	}
	n.SetRotationX(e.pitch * math32.Pi / 180)
	n.SetRotationY(-e.yaw * math32.Pi / 180)
	n.SetRotationZ(-e.roll * math32.Pi / 180)
	if e.lgtKind == 1 {
		gx, gy, gz := dirLightOffset(float64(e.pitch), float64(e.yaw))
		n.SetPosition(gx, gy, gz)
		w.syncVisualSun()
	}
}

// aimCameraEuler aims a camera with Blitz pitch/yaw: 0,0 looks +Z,
// positive pitch looks down. G3N Euler SetRotationX looks the wrong
// way (sky + a ground sliver) because G3N cameras face -Z.
func (w *World) aimCameraEuler(e *Entity) {
	n := e.node.GetNode()
	p := n.Position()
	bx, by, bz := fromG3N(p.X, p.Y, p.Z)
	pr := float64(e.pitch) * math.Pi / 180
	yr := float64(e.yaw) * math.Pi / 180
	cp := math.Cos(pr)
	dist := 24.0
	lx := float64(bx) + math.Sin(yr)*cp*dist
	ly := float64(by) - math.Sin(pr)*dist
	lz := float64(bz) + math.Cos(yr)*cp*dist
	gx, gy, gz := toG3N(float32(lx), float32(ly), float32(lz))
	up := math32.Vector3{0, 1, 0}
	n.LookAt(&math32.Vector3{gx, gy, gz}, &up)
}
