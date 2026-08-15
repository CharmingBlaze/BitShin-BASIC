#include "flecs.h"
#include <string.h>

typedef struct mb_val {
	float x, y, z, w;
	char s[128];
} mb_val;

ecs_entity_t mb_component(ecs_world_t *w, const char *name) {
	ecs_entity_desc_t ed;
	ecs_component_desc_t d;
	memset(&ed, 0, sizeof(ed));
	memset(&d, 0, sizeof(d));
	ed.name = name;
	d.entity = ecs_entity_init(w, &ed);
	d.type.size = (ecs_size_t)sizeof(mb_val);
	d.type.alignment = (ecs_size_t)8;
	return ecs_component_init(w, &d);
}

ecs_entity_t mb_entity(ecs_world_t *w, const char *name) {
	if (name && name[0]) {
		ecs_entity_desc_t d;
		memset(&d, 0, sizeof(d));
		d.name = name;
		return ecs_entity_init(w, &d);
	}
	return ecs_new(w);
}

void mb_set(ecs_world_t *w, ecs_entity_t e, ecs_entity_t c, const mb_val *v) {
	ecs_set_id(w, e, c, sizeof(mb_val), v);
}

const mb_val *mb_get(const ecs_world_t *w, ecs_entity_t e, ecs_entity_t c) {
	return (const mb_val *)ecs_get_id(w, e, c);
}

ecs_query_t *mb_query(ecs_world_t *w, const char *expr) {
	ecs_query_desc_t d;
	memset(&d, 0, sizeof(d));
	d.expr = expr;
	return ecs_query_init(w, &d);
}

int mb_query_collect(ecs_world_t *w, ecs_query_t *q, ecs_entity_t *out, int max) {
	ecs_iter_t it;
	int n = 0;
	int i;
	if (!q || !out || max <= 0) {
		return 0;
	}
	it = ecs_query_iter(w, q);
	while (ecs_query_next(&it)) {
		for (i = 0; i < it.count && n < max; i++) {
			out[n++] = it.entities[i];
		}
		if (n >= max) {
			ecs_iter_fini(&it);
			break;
		}
	}
	return n;
}

void mb_parent(ecs_world_t *w, ecs_entity_t child, ecs_entity_t parent) {
	ecs_add_id(w, child, ecs_pair(EcsChildOf, parent));
}

ecs_entity_t mb_get_parent(const ecs_world_t *w, ecs_entity_t e) {
	return ecs_get_target(w, e, EcsChildOf, 0);
}
