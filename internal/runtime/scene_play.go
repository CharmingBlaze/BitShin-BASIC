package runtime

import (
	"moonbasic/internal/phys3d"
	"moonbasic/internal/value"
)

// sceneJoint is the creation record for a constraint. Jolt constraints do not
// keep the pivot, axis, or limits, so a scene dump reads this instead.
type sceneJoint struct {
	id          int
	kind        string
	a, b        int
	v           []float32
	limited     bool
	hmin, hmax  float32
	hasFriction bool
	friction    float32
	hasMotor    bool
	motorVel    float32
	motorTorque float32
}

func (w *World) rememberJoint(id int, kind string, a, b int, v ...float32) {
	if id == 0 {
		return
	}
	w.sceneJoints = append(w.sceneJoints, sceneJoint{
		id: id, kind: kind, a: a, b: b, v: append([]float32(nil), v...),
	})
}

func (w *World) rememberCreateJoint(id, kind, a, b int, args []value.Value) {
	px := float32(argN(args, 3, 0))
	py := float32(argN(args, 4, 0))
	pz := float32(argN(args, 5, 0))
	switch kind {
	case phys3d.JointPoint:
		w.rememberJoint(id, "point", a, b, px, py, pz)
	case phys3d.JointSlider:
		w.rememberJoint(id, "slider", a, b, px, py, pz, float32(argN(args, 6, 1)), float32(argN(args, 7, 0)), float32(argN(args, 8, 0)))
	case phys3d.JointSpring:
		w.rememberJoint(id, "spring", a, b, px, py, pz, float32(argN(args, 6, 1)), float32(argN(args, 7, 8)), float32(argN(args, 8, 1)))
	case phys3d.JointFixed:
		w.rememberJoint(id, "fixed", a, b, px, py, pz)
	case phys3d.JointCone:
		w.rememberJoint(id, "cone", a, b, px, py, pz, float32(argN(args, 6, 0)), float32(argN(args, 7, 1)), float32(argN(args, 8, 0)), float32(argN(args, 9, 45)))
	case phys3d.JointSwingTwist:
		w.rememberJoint(id, "swing", a, b, px, py, pz, float32(argN(args, 6, 0)), float32(argN(args, 7, 1)), float32(argN(args, 8, 0)), float32(argN(args, 9, 45)), float32(argN(args, 10, 30)))
	default:
		w.rememberJoint(id, "hinge", a, b, px, py, pz, float32(argN(args, 6, 0)), float32(argN(args, 7, 1)), float32(argN(args, 8, 0)))
	}
}

func (w *World) forgetJoint(id int) {
	if id == 0 || len(w.sceneJoints) == 0 {
		return
	}
	kept := w.sceneJoints[:0]
	for _, j := range w.sceneJoints {
		if j.id != id {
			kept = append(kept, j)
		}
	}
	w.sceneJoints = kept
}

func (w *World) jointByID(id int) *sceneJoint {
	for i := range w.sceneJoints {
		if w.sceneJoints[i].id == id {
			return &w.sceneJoints[i]
		}
	}
	return nil
}

func (w *World) noteHingeLimits(id int, min, max float32) {
	if j := w.jointByID(id); j != nil {
		j.limited = true
		j.hmin, j.hmax = min, max
	}
}

func (w *World) noteHingeFriction(id int, friction float32) {
	if j := w.jointByID(id); j != nil {
		j.hasFriction = true
		j.friction = friction
	}
}

func (w *World) noteHingeMotor(id int, vel, torque float32) {
	if j := w.jointByID(id); j != nil {
		j.hasMotor = true
		j.motorVel, j.motorTorque = vel, torque
	}
}

func (w *World) sceneJointRows() []any {
	if len(w.sceneJoints) == 0 {
		return nil
	}
	out := make([]any, 0, len(w.sceneJoints))
	for _, j := range w.sceneJoints {
		row := map[string]any{"kind": j.kind, "a": j.a, "b": j.b, "v": append([]float32(nil), j.v...)}
		if j.limited {
			row["min"] = j.hmin
			row["max"] = j.hmax
		}
		if j.hasFriction {
			row["fric"] = j.friction
		}
		if j.hasMotor {
			row["mvel"] = j.motorVel
			row["mtorque"] = j.motorTorque
		}
		out = append(out, row)
	}
	return out
}

func (w *World) sceneController(id int) map[string]any {
	if c := w.fpsControllers[id]; c != nil {
		return map[string]any{
			"kind": "fps", "cam": c.camID, "h": c.height, "r": c.radius,
			"walk": c.walkSpeed, "sprint": c.sprintSpeed, "jump": c.jumpForce,
		}
	}
	if c := w.tpsControllers[id]; c != nil {
		return map[string]any{
			"kind": "tps", "cam": c.camID, "h": c.height, "r": c.radius,
			"move": c.moveSpeed, "sprint": c.sprintSpeed, "jump": c.jumpForce,
		}
	}
	if c := w.topDownControllers[id]; c != nil {
		return map[string]any{"kind": "topdown", "speed": c.speed, "turn": c.rotSpeed}
	}
	if c := w.platformerControllers[id]; c != nil {
		return map[string]any{"kind": "platformer", "speed": c.speed, "jump": c.jumpForce, "jumps": c.maxJumps}
	}
	c := w.vehCtrls[id]
	if c == nil {
		return nil
	}
	row := map[string]any{
		"kind": c.kind, "hx": c.hx, "hy": c.hy, "hz": c.hz, "mass": c.mass,
		"thrust": c.thrustMax, "cl": c.liftCl, "cd": c.dragCd, "stall": c.stall,
	}
	if w.phys3 != nil {
		if g := w.phys3.GetGravityScale(id); g != 1 {
			row["g"] = g
		}
		if d := w.phys3.GetLinearDamping(id); d > 0 {
			row["ld"] = d
		}
		if d := w.phys3.GetAngularDamping(id); d > 0 {
			row["ad"] = d
		}
	}
	return row
}

func (w *World) restoreSceneController(id int, em map[string]any, remap map[int]int) {
	raw, _ := em["ctrl"].(map[string]any)
	if raw == nil {
		return
	}
	kind, _ := raw["kind"].(string)
	if kind == "" || w.ents[id] == nil {
		return
	}
	w.ensurePhys3()
	switch kind {
	case "fps", "tps", "topdown", "platformer":
		w.restoreCharacterController(id, kind, raw, remap)
	default:
		w.restoreVehicleController(id, kind, raw)
	}
}

func (w *World) restoreCharacterController(id int, kind string, raw map[string]any, remap map[int]int) {
	px, py, pz := w.blitzPosOf(id)
	e := w.ents[id]
	switch kind {
	case "fps":
		if w.fpsControllers == nil {
			w.fpsControllers = map[int]*fpsCtrl{}
		}
		cam, _ := remapJointBody(remap, int(ctrlF(raw, "cam", 0)))
		c := &fpsCtrl{
			id: id, camID: cam,
			height: ctrlF(raw, "h", 1.8), radius: ctrlF(raw, "r", 0.4),
			walkSpeed: ctrlF(raw, "walk", 6), sprintSpeed: ctrlF(raw, "sprint", 10.5),
			jumpForce: ctrlF(raw, "jump", 9.5), crouchRatio: 1,
		}
		w.fpsControllers[id] = c
		w.phys3.AddCharacterController(id, px, py, pz, c.height, c.radius, 50, 80)
	case "tps":
		if w.tpsControllers == nil {
			w.tpsControllers = map[int]*tpsCtrl{}
		}
		cam, _ := remapJointBody(remap, int(ctrlF(raw, "cam", 0)))
		c := &tpsCtrl{
			id: id, camID: cam,
			height: ctrlF(raw, "h", 1.8), radius: ctrlF(raw, "r", 0.45),
			moveSpeed: ctrlF(raw, "move", 6.5), sprintSpeed: ctrlF(raw, "sprint", 11),
			jumpForce: ctrlF(raw, "jump", 10),
		}
		w.tpsControllers[id] = c
		w.phys3.AddCharacterController(id, px, py, pz, c.height, c.radius, 50, 80)
	case "topdown":
		if w.topDownControllers == nil {
			w.topDownControllers = map[int]*topDownCtrl{}
		}
		w.topDownControllers[id] = &topDownCtrl{
			id: id, speed: ctrlF(raw, "speed", 8), rotSpeed: ctrlF(raw, "turn", 18),
		}
		w.phys3.AddCharacterController(id, px, py, pz, 1.8, 0.45, 50, 80)
	case "platformer":
		jumps := int(ctrlF(raw, "jumps", 2))
		if jumps < 1 {
			jumps = 1
		}
		if w.platformerControllers == nil {
			w.platformerControllers = map[int]*platformerCtrl{}
		}
		w.platformerControllers[id] = &platformerCtrl{
			id: id, speed: ctrlF(raw, "speed", 8.5), jumpForce: ctrlF(raw, "jump", 12),
			maxJumps: jumps, jumpsLeft: jumps,
		}
		w.phys3.AddCharacterController(id, px, py, pz, 1.8, 0.4, 50, 80)
	}
	if e != nil {
		e.bodyType = 3
	}
}

func (w *World) restoreVehicleController(id int, kind string, raw map[string]any) {
	switch kind {
	case "car", "plane", "jet", "ship", "boat", "moto", "heli", "hover", "sub", "tank", "drone", "glider", "ski", "waterski", "mech", "lander", "jetski":
	default:
		return
	}
	hx := ctrlF(raw, "hx", 1)
	hy := ctrlF(raw, "hy", 0.5)
	hz := ctrlF(raw, "hz", 1)
	mass := ctrlF(raw, "mass", 1)
	args := []value.Value{
		value.Num(float64(id)),
		value.Num(float64(hx)),
		value.Num(float64(hy)),
		value.Num(float64(hz)),
		value.Num(float64(mass)),
	}
	c := w.bindCtrl(kind, args, hx, hy, hz, mass)
	c.thrustMax = ctrlF(raw, "thrust", 0)
	c.liftCl = ctrlF(raw, "cl", 0)
	c.dragCd = ctrlF(raw, "cd", 0)
	c.stall = ctrlF(raw, "stall", 0)
	if _, ok := raw["g"]; ok {
		w.phys3.SetGravityScale(id, ctrlF(raw, "g", 1))
	}
	if _, ok := raw["ld"]; ok {
		w.phys3.SetLinearDamping(id, ctrlF(raw, "ld", 0))
	}
	if _, ok := raw["ad"]; ok {
		w.phys3.SetAngularDamping(id, ctrlF(raw, "ad", 0))
	}
	switch kind {
	case "car":
		c.native = w.phys3.CreateWheeledVehicle(c.id, c.hx, c.hy, c.hz)
	case "moto":
		c.native = w.phys3.CreateMotorcycleVehicle(c.id, c.hx, c.hy, c.hz)
	case "tank":
		c.native = w.phys3.CreateTrackedVehicle(c.id, c.hx, c.hy, c.hz)
	case "plane":
		w.phys3.CreatePlaneController(c.id)
	}
}

func (w *World) restoreSceneMeshBody(id int, e *Entity, em map[string]any, shape, body int) {
	e.bodyType = body
	e.collKind = shape
	e.boxX = float32(anyToValue(em["hx"]).Number())
	e.boxY = float32(anyToValue(em["hy"]).Number())
	e.boxZ = float32(anyToValue(em["hz"]).Number())
	if rad := float32(anyToValue(em["rad"]).Number()); rad > 0 {
		e.radius = rad
	}
	if shape == 6 {
		return
	}
	if shape == 7 {
		w.addEntityMeshBody(id, e)
		return
	}
	motion := sceneBodyMotion(body)
	if w.addEntityConvexBody(id, e, motion) && motion == phys3d.MotionTypeDynamic {
		if mass := float32(anyToValue(em["mass"]).Number()); mass > 0 {
			w.phys3.SetMass(id, mass)
		}
	}
}

func (w *World) restoreSceneCompound(id int, em map[string]any) {
	if em == nil || int(anyToValue(em["shape"]).Number()) != 6 {
		return
	}
	e := w.ents[id]
	body := int(anyToValue(em["body"]).Number())
	if e == nil || body == 0 {
		return
	}
	w.ensurePhys3()
	parts, drop := w.gatherCompoundParts(id)
	if len(parts) == 0 {
		return
	}
	for _, cid := range drop {
		w.phys3.Remove(cid)
		if ce := w.ents[cid]; ce != nil {
			ce.bodyType = 0
			ce.collKind = 0
		}
	}
	px, py, pz := w.blitzPosOf(id)
	motion := sceneBodyMotion(body)
	w.phys3.AddCompound(id, parts, px, py, pz, motion)
	e.bodyType = body
	e.collKind = 6
	if motion == phys3d.MotionTypeDynamic {
		if mass := float32(anyToValue(em["mass"]).Number()); mass > 0 {
			w.phys3.SetMass(id, mass)
		}
	}
}

func sceneBodyMotion(body int) int {
	switch body {
	case 2:
		return phys3d.MotionTypeStatic
	case 3:
		return phys3d.MotionTypeKinematic
	default:
		return phys3d.MotionTypeDynamic
	}
}

func (w *World) restoreSceneJoints(root any, remap map[int]int) {
	m, _ := root.(map[string]any)
	if m == nil {
		return
	}
	list, _ := m["joints"].([]any)
	if len(list) == 0 {
		return
	}
	w.ensurePhys3()
	for _, raw := range list {
		row, _ := raw.(map[string]any)
		if row == nil {
			continue
		}
		kind, _ := row["kind"].(string)
		a, aOK := remapJointBody(remap, int(anyToValue(row["a"]).Number()))
		b, bOK := remapJointBody(remap, int(anyToValue(row["b"]).Number()))
		if !aOK || !bOK {
			continue
		}
		v := floatList(row["v"])
		id := w.createSceneJoint(kind, a, b, v)
		if id == 0 {
			continue
		}
		w.rememberJoint(id, kind, a, b, v...)
		if kind == "grab" {
			if w.grabs == nil {
				w.grabs = map[int]grabHold{}
			}
			w.grabs[a] = grabHold{joint: id, target: b}
		}
		if _, ok := row["min"]; ok {
			min := float32(anyToValue(row["min"]).Number())
			max := float32(anyToValue(row["max"]).Number())
			w.phys3.SetHingeLimits(id, min, max)
			w.noteHingeLimits(id, min, max)
		}
		if _, ok := row["fric"]; ok {
			fric := float32(anyToValue(row["fric"]).Number())
			w.phys3.SetHingeFriction(id, fric)
			w.noteHingeFriction(id, fric)
		}
		if _, ok := row["mvel"]; ok {
			vel := float32(anyToValue(row["mvel"]).Number())
			torque := float32(anyToValue(row["mtorque"]).Number())
			w.phys3.SetHingeMotor(id, vel, torque)
			w.noteHingeMotor(id, vel, torque)
		}
	}
}

func remapJointBody(remap map[int]int, id int) (int, bool) {
	if id == 0 {
		return 0, true
	}
	n := remap[id]
	return n, n != 0
}

func (w *World) createSceneJoint(kind string, a, b int, v []float32) int {
	at := func(i int) float32 {
		if i < 0 || i >= len(v) {
			return 0
		}
		return v[i]
	}
	switch kind {
	case "point":
		if len(v) < 3 {
			return 0
		}
		return w.phys3.CreatePointJoint(a, b, at(0), at(1), at(2))
	case "slider":
		if len(v) < 6 {
			return 0
		}
		return w.phys3.CreateSliderJoint(a, b, at(0), at(1), at(2), at(3), at(4), at(5))
	case "spring":
		if len(v) < 6 {
			return 0
		}
		return w.phys3.CreateSpringJoint(a, b, at(0), at(1), at(2), at(3), at(4), at(5))
	case "fixed":
		if len(v) < 3 {
			return 0
		}
		return w.phys3.CreateFixedJoint(a, b, at(0), at(1), at(2))
	case "cone":
		if len(v) < 7 {
			return 0
		}
		return w.phys3.CreateConeJoint(a, b, at(0), at(1), at(2), at(3), at(4), at(5), at(6))
	case "swing":
		if len(v) < 8 {
			return 0
		}
		return w.phys3.CreateSwingTwistJoint(a, b, at(0), at(1), at(2), at(3), at(4), at(5), at(6), at(7))
	case "grab":
		if len(v) < 5 {
			return 0
		}
		return w.phys3.CreateGrabJoint(a, b, at(0), at(1), at(2), at(3), at(4))
	default:
		if len(v) < 6 {
			return 0
		}
		return w.phys3.CreateHingeJoint(a, b, at(0), at(1), at(2), at(3), at(4), at(5))
	}
}

func sceneMaterial(e *Entity) map[string]any {
	if e == nil {
		return nil
	}
	if e.pbrWrap == nil && e.albedoPath == "" {
		return nil
	}
	row := map[string]any{}
	if p := e.pbrWrap; p != nil {
		row["metal"] = p.metallic
		row["rough"] = p.roughness
		row["ao"] = p.ao
		row["ar"] = p.albedo.R
		row["ag"] = p.albedo.G
		row["ab"] = p.albedo.B
		row["aa"] = p.albedo.A
		row["er"] = p.emissive.R
		row["eg"] = p.emissive.G
		row["eb"] = p.emissive.B
		row["ibl"] = p.ibl
	}
	if e.albedoPath != "" {
		row["tex"] = e.albedoPath
	}
	return row
}

func sceneSkinJoints(e *Entity) []any {
	if e == nil || e.skin == nil || len(e.skin.joints) == 0 {
		return nil
	}
	out := make([]any, 0, len(e.skin.joints))
	for _, j := range e.skin.joints {
		x, y, z := j.bindX, j.bindY, j.bindZ
		if j.node != nil {
			p := j.node.Position()
			x, y, z = fromG3N(p.X, p.Y, p.Z)
		}
		out = append(out, map[string]any{"name": j.name, "x": x, "y": y, "z": z})
	}
	return out
}

func (w *World) restoreSceneMaterial(e *Entity, em map[string]any) {
	raw, _ := em["mat"].(map[string]any)
	if e == nil || raw == nil {
		return
	}
	if _, ok := raw["metal"]; ok {
		p := w.convertEntityPBR(e)
		if p != nil {
			p.metallic = ctrlF(raw, "metal", p.metallic)
			p.roughness = ctrlF(raw, "rough", p.roughness)
			p.ao = ctrlF(raw, "ao", p.ao)
			p.albedo.R = ctrlF(raw, "ar", p.albedo.R)
			p.albedo.G = ctrlF(raw, "ag", p.albedo.G)
			p.albedo.B = ctrlF(raw, "ab", p.albedo.B)
			p.albedo.A = ctrlF(raw, "aa", p.albedo.A)
			if p.albedo.A <= 0 {
				p.albedo.A = 1
			}
			p.emissive.R = ctrlF(raw, "er", p.emissive.R)
			p.emissive.G = ctrlF(raw, "eg", p.emissive.G)
			p.emissive.B = ctrlF(raw, "eb", p.emissive.B)
			p.ibl = int(ctrlF(raw, "ibl", float32(p.ibl)))
			p.applyFactors()
		}
	}
	path, _ := raw["tex"].(string)
	if path == "" {
		return
	}
	e.albedoPath = path
	id, err := w.loadTextureFile(path)
	if err != nil || id == 0 {
		return
	}
	t := w.texs[id]
	if t == nil {
		return
	}
	e.albedoPath = t.path
	e.albedoTex = t.tex
	if e.mat != nil {
		e.mat.AddTexture(t.tex)
	}
	if e.pbr != nil {
		e.pbr.SetBaseColorMap(t.tex)
	}
}

func (w *World) restoreSceneSkin(e *Entity, em map[string]any) {
	if e == nil || e.skin == nil {
		return
	}
	list, _ := em["joints"].([]any)
	if len(list) == 0 {
		return
	}
	by := map[string]*skinJoint{}
	for i := range e.skin.joints {
		by[e.skin.joints[i].name] = &e.skin.joints[i]
	}
	for _, raw := range list {
		row, _ := raw.(map[string]any)
		if row == nil {
			continue
		}
		name, _ := row["name"].(string)
		j := by[name]
		if j == nil || j.node == nil {
			continue
		}
		x := float32(anyToValue(row["x"]).Number())
		y := float32(anyToValue(row["y"]).Number())
		z := float32(anyToValue(row["z"]).Number())
		gx, gy, gz := toG3N(x, y, z)
		j.node.SetPosition(gx, gy, gz)
	}
}

func ctrlF(m map[string]any, key string, def float32) float32 {
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	return float32(anyToValue(v).Number())
}

func floatList(v any) []float32 {
	switch n := v.(type) {
	case []float32:
		return append([]float32(nil), n...)
	case []any:
		out := make([]float32, 0, len(n))
		for _, item := range n {
			out = append(out, float32(anyToValue(item).Number()))
		}
		return out
	default:
		return nil
	}
}
