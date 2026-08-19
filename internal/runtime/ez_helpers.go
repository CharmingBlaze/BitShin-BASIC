package runtime

import (
	"math"
	"strings"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/value"
)

type grabHold struct {
	joint, target int
	parented      bool
	prevBody      int
}

type projFly struct {
	vx, vy, vz float32
	gravity    float32
	life       float32
	radius     float32
	bounce     float32
	impulse    float32
	ignore     int
}

type beamLink struct {
	a, b  int
	width float32
}

type boneAttach struct {
	child, mesh int
	bone        string
}

func entityBlitzForward(e *Entity) (float32, float32, float32) {
	if e == nil {
		return 0, 0, 1
	}
	pr := float64(e.pitch) * math.Pi / 180
	yr := float64(e.yaw) * math.Pi / 180
	cp := math.Cos(pr)
	return float32(math.Sin(yr) * cp), float32(-math.Sin(pr)), float32(math.Cos(yr) * cp)
}

func (w *World) blitzPosOf(id int) (float32, float32, float32) {
	e := w.ents[id]
	if e == nil || e.node == nil {
		return 0, 0, 0
	}
	w.refreshWorldMatrices()
	p := worldPos(e.node.GetNode())
	return fromG3N(p.X, p.Y, p.Z)
}

func findNamedNode(n core.INode, name string) core.INode {
	if n == nil {
		return nil
	}
	want := strings.ToLower(name)
	gn := n.GetNode()
	if gn != nil && strings.ToLower(gn.Name()) == want {
		return n
	}
	if gn == nil {
		return nil
	}
	for _, c := range gn.Children() {
		if found := findNamedNode(c, name); found != nil {
			return found
		}
	}
	return nil
}

func (w *World) dropGrab(holder int) {
	g, ok := w.grabs[holder]
	if !ok {
		return
	}
	if g.joint != 0 && w.phys3 != nil {
		w.phys3.RemoveJoint(g.joint)
	}
	if g.parented {
		if te := w.ents[g.target]; te != nil {
			te.parent = 0
			if te.node != nil && w.scene != nil {
				if p := te.node.GetNode().Parent(); p != nil {
					p.GetNode().Remove(te.node)
				}
				w.scene.Add(te.node)
			}
			if g.prevBody != 0 {
				te.bodyType = g.prevBody
			}
		}
	}
	delete(w.grabs, holder)
}

func (w *World) grabTarget(holder, target int, freq, damp float32, px, py, pz float32, anchored bool) int {
	if holder == 0 || target == 0 || holder == target {
		return 0
	}
	w.ensurePhys3()
	w.dropGrab(holder)
	if !anchored {
		px, py, pz = w.blitzPosOf(target)
	}
	if freq <= 0 {
		freq = 8
	}
	if damp < 0 {
		damp = 1
	}
	joint := w.phys3.CreateGrabJoint(holder, target, px, py, pz, freq, damp)
	hold := grabHold{joint: joint, target: target}
	if joint == 0 {
		child, err := w.ent(target)
		if err == nil && child.node != nil {
			hold.parented = true
			hold.prevBody = child.bodyType
			if p := child.node.GetNode().Parent(); p != nil {
				p.GetNode().Remove(child.node)
			}
			w.parentNode(holder).GetNode().Add(child.node)
			child.parent = holder
			child.bodyType = 3
		}
	}
	if w.grabs == nil {
		w.grabs = map[int]grabHold{}
	}
	w.grabs[holder] = hold
	return target
}

func (w *World) tickProjectiles(dt float32) {
	if len(w.projectiles) == 0 {
		return
	}
	alive := map[int]*projFly{}
	for id, p := range w.projectiles {
		e := w.ents[id]
		if e == nil || p == nil {
			continue
		}
		p.life -= dt
		if p.life <= 0 {
			continue
		}
		ox, oy, oz := w.blitzPosOf(id)
		if e.bodyType != 0 && w.phys3 != nil {
			if p.gravity != 0 {
				p.vy += p.gravity * dt
				w.phys3.SetVelocity(id, p.vx, p.vy, p.vz)
			}
			if vx, vy, vz, ok := w.phys3.GetVelocity(id); ok {
				p.vx, p.vy, p.vz = vx, vy, vz
			}
			if px, py, pz, ok := w.phys3.GetPosition(id); ok {
				ox, oy, oz = px, py, pz
			}
		} else {
			p.vy += p.gravity * dt
		}
		dx, dy, dz := p.vx*dt, p.vy*dt, p.vz*dt
		hitID, hx, hy, hz, ok := 0, float32(0), float32(0), float32(0), false
		if w.phys3 != nil {
			hitID, hx, hy, hz, ok = w.phys3.Raycast(ox, oy, oz, dx, dy, dz)
		}
		if ok && hitID != 0 && hitID != id && hitID != p.ignore {
			w.pickID, w.pickX, w.pickY, w.pickZ = hitID, hx, hy, hz
			if p.impulse != 0 && w.phys3 != nil {
				w.phys3.ApplyImpulse(hitID, p.vx*p.impulse, p.vy*p.impulse, p.vz*p.impulse)
			}
			if p.bounce > 0 {
				p.vx *= -p.bounce
				p.vy *= -p.bounce
				p.vz *= -p.bounce
				gx, gy, gz := toG3N(hx, hy, hz)
				e.node.GetNode().SetPosition(gx, gy, gz)
				if e.bodyType != 0 && w.phys3 != nil {
					w.phys3.SetPosition(id, hx, hy, hz)
					w.phys3.SetVelocity(id, p.vx, p.vy, p.vz)
				}
				alive[id] = p
				continue
			}
			gx, gy, gz := toG3N(hx, hy, hz)
			e.node.GetNode().SetPosition(gx, gy, gz)
			if e.bodyType != 0 && w.phys3 != nil {
				w.phys3.SetPosition(id, hx, hy, hz)
				w.phys3.SetVelocity(id, 0, 0, 0)
			}
			continue
		}
		if e.bodyType == 0 || w.phys3 == nil {
			nx, ny, nz := ox+dx, oy+dy, oz+dz
			gx, gy, gz := toG3N(nx, ny, nz)
			e.node.GetNode().SetPosition(gx, gy, gz)
		}
		alive[id] = p
	}
	w.projectiles = alive
}

func (w *World) tickBeams() {
	if len(w.beams) == 0 {
		return
	}
	w.refreshWorldMatrices()
	for id, b := range w.beams {
		e := w.ents[id]
		ae := w.ents[b.a]
		be := w.ents[b.b]
		if e == nil || ae == nil || be == nil || e.node == nil {
			continue
		}
		pa := worldPos(ae.node.GetNode())
		pb := worldPos(be.node.GetNode())
		mx := (pa.X + pb.X) * 0.5
		my := (pa.Y + pb.Y) * 0.5
		mz := (pa.Z + pb.Z) * 0.5
		dx, dy, dz := pb.X-pa.X, pb.Y-pa.Y, pb.Z-pa.Z
		length := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
		if length < 0.02 {
			length = 0.02
		}
		n := e.node.GetNode()
		n.SetPosition(mx, my, mz)
		up := math32.Vector3{0, 1, 0}
		n.LookAt(&pb, &up)
		wid := b.width
		n.SetScale(wid, wid, length)
	}
}

func (w *World) tickBoneAttaches() {
	for _, a := range w.boneAttaches {
		child := w.ents[a.child]
		mesh := w.ents[a.mesh]
		if child == nil || mesh == nil || child.node == nil || mesh.node == nil {
			continue
		}
		bone := findNamedNode(mesh.node, a.bone)
		if bone == nil {
			continue
		}
		cn := child.node.GetNode()
		if cn.Parent() == bone.GetNode() {
			continue
		}
		if p := cn.Parent(); p != nil {
			p.GetNode().Remove(child.node)
		}
		bone.GetNode().Add(child.node)
		child.parent = a.mesh
	}
}

func (w *World) ezHelperCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"grab": need(func(a []value.Value) (value.Value, error) {
			anchored := len(a) >= 7
			px, py, pz := float32(argN(a, 4, 0)), float32(argN(a, 5, 0)), float32(argN(a, 6, 0))
			id := w.grabTarget(argI(a, 0, 0), argI(a, 1, 0), float32(argN(a, 2, 8)), float32(argN(a, 3, 1)), px, py, pz, anchored)
			return value.Num(float64(id)), nil
		}),
		"grabpick": need(func(a []value.Value) (value.Value, error) {
			holder := argI(a, 0, 0)
			maxDist := argN(a, 1, 8)
			if maxDist <= 0 {
				maxDist = 8
			}
			e, err := w.ent(holder)
			if err != nil {
				return value.Value{}, err
			}
			w.ensurePhys3()
			ox, oy, oz := w.blitzPosOf(holder)
			fx, fy, fz := entityBlitzForward(e)
			hit, hx, hy, hz, ok := w.phys3.Raycast(ox, oy, oz, fx*float32(maxDist), fy*float32(maxDist), fz*float32(maxDist))
			if !ok || hit == 0 || hit == holder {
				return value.Num(0), nil
			}
			w.pickID, w.pickX, w.pickY, w.pickZ = hit, hx, hy, hz
			id := w.grabTarget(holder, hit, float32(argN(a, 2, 8)), float32(argN(a, 3, 1)), hx, hy, hz, true)
			return value.Num(float64(id)), nil
		}),
		"dropgrab": n(func(a []value.Value) (value.Value, error) {
			w.dropGrab(argI(a, 0, 0))
			return z()
		}),
		"throw": need(func(a []value.Value) (value.Value, error) {
			holder := argI(a, 0, 0)
			speed := float32(argN(a, 1, 12))
			g, ok := w.grabs[holder]
			target := g.target
			w.dropGrab(holder)
			if !ok || target == 0 {
				return value.Num(0), nil
			}
			he := w.ents[holder]
			fx, fy, fz := entityBlitzForward(he)
			w.ensurePhys3()
			w.phys3.SetVelocity(target, fx*speed, fy*speed, fz*speed)
			w.phys3.ApplyImpulse(target, fx*speed, fy*speed, fz*speed)
			return value.Num(float64(target)), nil
		}),
		"grabbedentity": n(func(a []value.Value) (value.Value, error) {
			g := w.grabs[argI(a, 0, 0)]
			return value.Num(float64(g.target)), nil
		}),
		"createprojectile": need(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			e, err := w.ent(id)
			if err != nil {
				return value.Value{}, err
			}
			speed := float32(argN(a, 1, 20))
			fx, fy, fz := entityBlitzForward(e)
			if w.projectiles == nil {
				w.projectiles = map[int]*projFly{}
			}
			life := float32(argN(a, 3, 4))
			if life <= 0 {
				life = 4
			}
			w.projectiles[id] = &projFly{
				vx: fx * speed, vy: fy * speed, vz: fz * speed,
				gravity: float32(argN(a, 2, 0)),
				life:    life,
				radius:  float32(argN(a, 4, 0.1)),
				bounce:  float32(argN(a, 5, 0)),
				impulse: float32(argN(a, 6, 1)),
				ignore:  argI(a, 7, 0),
			}
			w.ensurePhys3()
			if e.bodyType != 0 {
				w.phys3.SetCCD(id, true)
				w.phys3.SetGravityScale(id, 0)
				w.phys3.SetVelocity(id, fx*speed, fy*speed, fz*speed)
			}
			return value.Num(float64(id)), nil
		}),
		"createbeam": need(func(a []value.Value) (value.Value, error) {
			src, dst := argI(a, 0, 0), argI(a, 1, 0)
			if _, err := w.ent(src); err != nil {
				return value.Value{}, err
			}
			if _, err := w.ent(dst); err != nil {
				return value.Value{}, err
			}
			width := float32(argN(a, 2, 0.08))
			if width <= 0 {
				width = 0.08
			}
			id := w.meshEnt(geometry.NewCube(1), 0)
			if w.beams == nil {
				w.beams = map[int]*beamLink{}
			}
			w.beams[id] = &beamLink{a: src, b: dst, width: width}
			w.tickBeams()
			return value.Num(float64(id)), nil
		}),
		"placeatray": need(func(a []value.Value) (value.Value, error) {
			src := argI(a, 0, 0)
			dst := argI(a, 1, 0)
			maxDist := float32(argN(a, 2, 40))
			se, err := w.ent(src)
			if err != nil {
				return value.Value{}, err
			}
			de, err := w.ent(dst)
			if err != nil {
				return value.Value{}, err
			}
			if maxDist <= 0 {
				maxDist = 40
			}
			w.ensurePhys3()
			ox, oy, oz := w.blitzPosOf(src)
			fx, fy, fz := entityBlitzForward(se)
			hit, hx, hy, hz, ok := w.phys3.Raycast(ox, oy, oz, fx*maxDist, fy*maxDist, fz*maxDist)
			if !ok || hit == src {
				hx, hy, hz = ox+fx*maxDist, oy+fy*maxDist, oz+fz*maxDist
				hit = 0
			}
			w.pickID, w.pickX, w.pickY, w.pickZ = hit, hx, hy, hz
			gx, gy, gz := toG3N(hx, hy, hz)
			de.node.GetNode().SetPosition(gx, gy, gz)
			if de.bodyType != 0 {
				w.phys3.SetPosition(dst, hx, hy, hz)
			}
			return value.Num(float64(hit)), nil
		}),
		"attachtobone": need(func(a []value.Value) (value.Value, error) {
			childID := argI(a, 0, 0)
			meshID := argI(a, 1, 0)
			bone := argS(a, 2)
			child, err := w.ent(childID)
			if err != nil {
				return value.Value{}, err
			}
			mesh, err := w.ent(meshID)
			if err != nil {
				return value.Value{}, err
			}
			target := findNamedNode(mesh.node, bone)
			if target == nil {
				for _, other := range w.ents {
					if other != nil && other.parent == meshID && strings.EqualFold(other.name, bone) && other.node != nil {
						target = other.node
						break
					}
				}
			}
			if target == nil {
				target = mesh.node
			}
			if p := child.node.GetNode().Parent(); p != nil {
				p.GetNode().Remove(child.node)
			}
			target.GetNode().Add(child.node)
			child.parent = meshID
			w.boneAttaches = append(w.boneAttaches, boneAttach{child: childID, mesh: meshID, bone: bone})
			return value.Num(float64(childID)), nil
		}),
	}
}
