// Package flecs wraps Flecs 4.1.6 (Sander Mertens C amalgamation).
// github.com/SanderMertens/flecs-go is not published upstream; this module
// pins that import path to the in-tree C sources.
package flecs

/*
#cgo CFLAGS: -I${SRCDIR} -std=c99 -O2 -DFLECS_NDEBUG -DFLECS_CUSTOM_BUILD -DFLECS_SYSTEM -DFLECS_PIPELINE -DFLECS_TIMER -DFLECS_OS_API_IMPL -DFLECS_MODULE -DFLECS_PARSER -DFLECS_QUERY_DSL -DFLECS_LOG
#cgo windows LDFLAGS: -lws2_32 -ldbghelp
#include "flecs.h"
#include <stdlib.h>
#include <string.h>

typedef struct mb_val {
	float x, y, z, w;
	char s[128];
} mb_val;

extern ecs_entity_t mb_component(ecs_world_t *w, const char *name);
extern ecs_entity_t mb_entity(ecs_world_t *w, const char *name);
extern void mb_set(ecs_world_t *w, ecs_entity_t e, ecs_entity_t c, const mb_val *v);
extern const mb_val *mb_get(const ecs_world_t *w, ecs_entity_t e, ecs_entity_t c);
extern ecs_query_t *mb_query(ecs_world_t *w, const char *expr);
extern int mb_query_collect(ecs_world_t *w, ecs_query_t *q, ecs_entity_t *out, int max);
extern void mb_parent(ecs_world_t *w, ecs_entity_t child, ecs_entity_t parent);
extern ecs_entity_t mb_get_parent(const ecs_world_t *w, ecs_entity_t e);
*/
import "C"
import (
	"unsafe"
)

const Version = "4.1.6"

type World struct {
	ptr *C.ecs_world_t
}

type Entity uint64
type Component uint64

type Query struct {
	ptr   *C.ecs_query_t
	world *World
}

type Value struct {
	X, Y, Z, W float32
	S          string
}

func NewWorld() *World {
	return &World{ptr: C.ecs_init()}
}

func (w *World) Fini() {
	if w == nil || w.ptr == nil {
		return
	}
	C.ecs_fini(w.ptr)
	w.ptr = nil
}

func (w *World) Progress(dt float32) bool {
	if w == nil || w.ptr == nil {
		return false
	}
	return bool(C.ecs_progress(w.ptr, C.ecs_ftime_t(dt)))
}

func (w *World) Entity(name string) Entity {
	if w == nil || w.ptr == nil {
		return 0
	}
	var cname *C.char
	if name != "" {
		cname = C.CString(name)
		defer C.free(unsafe.Pointer(cname))
	}
	return Entity(C.mb_entity(w.ptr, cname))
}

func (w *World) Component(name string) Component {
	if w == nil || w.ptr == nil || name == "" {
		return 0
	}
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	return Component(C.mb_component(w.ptr, cname))
}

func (w *World) Set(e Entity, c Component, v Value) {
	if w == nil || w.ptr == nil || e == 0 || c == 0 {
		return
	}
	var cv C.mb_val
	cv.x = C.float(v.X)
	cv.y = C.float(v.Y)
	cv.z = C.float(v.Z)
	cv.w = C.float(v.W)
	if v.S != "" {
		cs := C.CString(v.S)
		C.strncpy(&cv.s[0], cs, 127)
		C.free(unsafe.Pointer(cs))
	}
	C.mb_set(w.ptr, C.ecs_entity_t(e), C.ecs_entity_t(c), &cv)
}

func (w *World) Get(e Entity, c Component) (Value, bool) {
	if w == nil || w.ptr == nil {
		return Value{}, false
	}
	p := C.mb_get(w.ptr, C.ecs_entity_t(e), C.ecs_entity_t(c))
	if p == nil {
		return Value{}, false
	}
	return Value{
		X: float32(p.x),
		Y: float32(p.y),
		Z: float32(p.z),
		W: float32(p.w),
		S: C.GoString(&p.s[0]),
	}, true
}

func (w *World) Has(e Entity, c Component) bool {
	if w == nil || w.ptr == nil {
		return false
	}
	return bool(C.ecs_has_id(w.ptr, C.ecs_entity_t(e), C.ecs_id_t(c)))
}

func (w *World) Add(e Entity, c Component) {
	if w == nil || w.ptr == nil || e == 0 || c == 0 {
		return
	}
	C.ecs_add_id(w.ptr, C.ecs_entity_t(e), C.ecs_id_t(c))
}

func (w *World) Remove(e Entity, c Component) {
	if w == nil || w.ptr == nil {
		return
	}
	C.ecs_remove_id(w.ptr, C.ecs_entity_t(e), C.ecs_id_t(c))
}

func (w *World) Delete(e Entity) {
	if w == nil || w.ptr == nil || e == 0 {
		return
	}
	C.ecs_delete(w.ptr, C.ecs_entity_t(e))
}

func (w *World) Alive(e Entity) bool {
	if w == nil || w.ptr == nil {
		return false
	}
	return bool(C.ecs_is_alive(w.ptr, C.ecs_entity_t(e)))
}

func (w *World) Valid(e Entity) bool {
	if w == nil || w.ptr == nil {
		return false
	}
	return bool(C.ecs_is_valid(w.ptr, C.ecs_entity_t(e)))
}

func (w *World) Lookup(name string) Entity {
	if w == nil || w.ptr == nil || name == "" {
		return 0
	}
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	return Entity(C.ecs_lookup(w.ptr, cname))
}

func (w *World) Name(e Entity) string {
	if w == nil || w.ptr == nil {
		return ""
	}
	p := C.ecs_get_name(w.ptr, C.ecs_entity_t(e))
	if p == nil {
		return ""
	}
	return C.GoString(p)
}

func (w *World) SetName(e Entity, name string) {
	if w == nil || w.ptr == nil || e == 0 {
		return
	}
	var cname *C.char
	if name != "" {
		cname = C.CString(name)
		defer C.free(unsafe.Pointer(cname))
	}
	C.ecs_set_name(w.ptr, C.ecs_entity_t(e), cname)
}

func (w *World) Count(c Component) int {
	if w == nil || w.ptr == nil || c == 0 {
		return 0
	}
	return int(C.ecs_count_id(w.ptr, C.ecs_id_t(c)))
}

func (w *World) Query(expr string) *Query {
	if w == nil || w.ptr == nil {
		return nil
	}
	var cexpr *C.char
	if expr != "" {
		cexpr = C.CString(expr)
		defer C.free(unsafe.Pointer(cexpr))
	}
	q := C.mb_query(w.ptr, cexpr)
	if q == nil {
		return nil
	}
	return &Query{ptr: q, world: w}
}

func (q *Query) Each() []Entity {
	if q == nil || q.ptr == nil || q.world == nil || q.world.ptr == nil {
		return nil
	}
	buf := make([]C.ecs_entity_t, 16384)
	n := int(C.mb_query_collect(q.world.ptr, q.ptr, &buf[0], C.int(len(buf))))
	if n <= 0 {
		return nil
	}
	out := make([]Entity, n)
	for i := 0; i < n; i++ {
		out[i] = Entity(buf[i])
	}
	return out
}

func (w *World) SetParent(child, parent Entity) {
	if w == nil || w.ptr == nil || child == 0 {
		return
	}
	C.mb_parent(w.ptr, C.ecs_entity_t(child), C.ecs_entity_t(parent))
}

func (w *World) Parent(e Entity) Entity {
	if w == nil || w.ptr == nil {
		return 0
	}
	return Entity(C.mb_get_parent(w.ptr, C.ecs_entity_t(e)))
}
