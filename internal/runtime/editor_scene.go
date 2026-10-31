package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/g3n/engine/math32"
)

// The editor's view of entity state for undo, and of scene files. Undo
// snapshots are scene v2 rows (entityJSON), so restoring brings back the
// material, body, light, tags, and script too. The node's own transform is
// kept alongside, so restored positions are exact, and kinds the scene
// format cannot hold (terrain, water) still undo their transform.

func (w *World) editorState(id int) map[string]any {
	e := w.ents[id]
	if e == nil {
		return nil
	}
	st := w.editorRaw(id, e)
	if row := w.entityJSON(id); row != nil {
		st["v2"] = row
	}
	return st
}

func (w *World) editorRaw(id int, e *Entity) map[string]any {
	n := e.node.GetNode()
	p, q, s := n.Position(), n.Quaternion(), n.Scale()
	st := map[string]any{
		"pos":     [3]float32{p.X, p.Y, p.Z},
		"quat":    [4]float32{q.X, q.Y, q.Z, q.W},
		"scale":   [3]float32{s.X, s.Y, s.Z},
		"euler":   [3]float32{e.pitch, e.yaw, e.roll},
		"name":    e.name,
		"visible": n.Visible(),
		"parent":  e.parent,
		"tint":    e.tint,
	}
	if pm := e.pbrWrap; pm != nil {
		st["pbr"] = [5]float32{pm.metallic, pm.roughness, pm.emissive.R, pm.emissive.G, pm.emissive.B}
		st["shading"] = pm.shading
	}
	return st
}

func (w *World) applyEditorState(id int, st map[string]any) {
	e := w.ents[id]
	if e == nil || st == nil {
		return
	}
	if row, ok := st["v2"].(map[string]any); ok {
		if e.meta != nil && e.meta.uid != "" {
			w.setEntityUID(id, e.meta.uid) // a soft delete may have dropped it from the index
		}
		if nid, err := w.applyEntityJSON(row, true); err == nil && nid != id {
			return // rebuilt under a new id (kind change): its row is complete
		}
		e = w.ents[id]
		if e == nil {
			return
		}
	}
	n := e.node.GetNode()
	if pid, ok := st["parent"].(int); ok && pid != e.parent {
		if p := n.Parent(); p != nil {
			p.GetNode().Remove(e.node)
		}
		w.parentNode(pid).GetNode().Add(e.node)
		e.parent = pid
	}
	if p, ok := st["pos"].([3]float32); ok {
		n.SetPosition(p[0], p[1], p[2])
	}
	if q, ok := st["quat"].([4]float32); ok {
		n.SetQuaternion(q[0], q[1], q[2], q[3])
	}
	if s, ok := st["scale"].([3]float32); ok {
		n.SetScale(s[0], s[1], s[2])
	}
	if eu, ok := st["euler"].([3]float32); ok {
		e.pitch, e.yaw, e.roll = eu[0], eu[1], eu[2]
	}
	if name, ok := st["name"].(string); ok {
		e.name = name
		n.SetName(name)
	}
	if v, ok := st["visible"].(bool); ok {
		n.SetVisible(v)
	}
	if t, ok := st["tint"].(math32.Color4); ok {
		w.setEntityRGB(e, t.R*255, t.G*255, t.B*255)
		e.tint.A = t.A
		if e.mat != nil {
			e.mat.SetOpacity(t.A)
		}
	}
	if pb, ok := st["pbr"].([5]float32); ok {
		w.eachEntityPBR(e, func(m *pbrMat) {
			m.metallic, m.roughness = pb[0], pb[1]
			m.emissive = math32.Color{R: pb[2], G: pb[3], B: pb[4]}
			if sh, ok := st["shading"].(int); ok {
				m.shading = sh
			}
			m.applyFactors()
		})
	}
}

// reparentKeepWorld moves child under parent (0 is the scene) without
// moving it on screen.
func (w *World) reparentKeepWorld(child, parent int) error {
	e := w.ents[child]
	if e == nil {
		return fmt.Errorf("no entity %d", child)
	}
	n := e.node.GetNode()
	n.UpdateMatrixWorld()
	world := n.MatrixWorld()
	pn := w.parentNode(parent).GetNode()
	pn.UpdateMatrixWorld()
	pw := pn.MatrixWorld()
	var inv math32.Matrix4
	if err := inv.GetInverse(&pw); err != nil {
		return err
	}
	var local math32.Matrix4
	local.MultiplyMatrices(&inv, &world)
	if p := n.Parent(); p != nil {
		p.GetNode().Remove(e.node)
	}
	pn.Add(e.node)
	e.parent = parent
	var pos, scale math32.Vector3
	var q math32.Quaternion
	local.Decompose(&pos, &q, &scale)
	n.SetPositionVec(&pos)
	n.SetQuaternionQuat(&q)
	n.SetScaleVec(&scale)
	w.syncEulerFromNode(e)
	return nil
}

// syncEulerFromNode keeps Blitz pitch/yaw/roll in step with a rotation set
// on the node directly (gizmo, reparent). applyRot sets X=pitch, Y=-yaw,
// Z=-roll in G3N's XYZ order; this is its inverse.
func (w *World) syncEulerFromNode(e *Entity) {
	if e == nil || e.cam != nil {
		return
	}
	r := e.node.GetNode().Rotation()
	e.pitch = r.X * 180 / math32.Pi
	e.yaw = -r.Y * 180 / math32.Pi
	e.roll = -r.Z * 180 / math32.Pi
}

func (w *World) editorLoadScene(path string) error {
	root, err := w.readJSONFile(path)
	if err != nil {
		return err
	}
	m, ok := root.(map[string]any)
	if !ok {
		return fmt.Errorf("%s is not a scene object", filepath.Base(path))
	}
	return w.loadSceneV2(m, false)
}

func (w *World) editorSaveScene(path string) error {
	b, err := json.MarshalIndent(w.sceneV2(), "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, b, 0o644)
}
