extends RefCounted

const UFA_SEED := 45001
const NAMES := {"site":"Свободная площадка","works":"Стройка","bakery": "Пекарня", "near": "Ближняя мельница", "remote": "Дальняя мельница", "homes": "Жилой дом", "park": "Сквер"}

static func streets(rng: RandomNumberGenerator) -> PackedInt32Array:
 var lines := PackedInt32Array([0])
 for i in range(5):
  lines.append(lines[i] + rng.randi_range(11, 15))
 var center := floori(lines[5] / 2.0)
 for i in range(lines.size()):
  lines[i] -= center
 return lines

static func generate(district_seed: int, bakery_world: bool = true) -> Dictionary:
 var rng := RandomNumberGenerator.new()
 rng.seed = district_seed
 var xs := streets(rng)
 var zs := streets(rng)
 var near_cell := Vector2i(2 + (1 if rng.randi_range(0,1) else -1), 2)
 var far_cell := Vector2i(4 if rng.randi_range(0,1) else 0, 4 if rng.randi_range(0,1) else 0)
 var lots: Array[Dictionary] = []
 for x in range(5):
  for z in range(5):
   var cell := Vector2i(x,z)
   var kind := "park" if rng.randi_range(0,4) == 0 else "homes"
   if bakery_world and cell == Vector2i(2,2):
    kind = "bakery"
   elif bakery_world and cell == near_cell:
    kind = "near"
   elif bakery_world and cell == far_cell:
    kind = "remote"
   var origin := Vector2(xs[x]+1.25,zs[z]+1.25)
   var end := Vector2(xs[x+1]-1.25,zs[z+1]-1.25)
   lots.append({"kind": kind, "plot": Rect2(origin,end-origin), "height": rng.randf_range(2.6,5.4)})
 return {"xs": xs, "zs": zs, "lots": lots}
