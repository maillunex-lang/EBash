extends Node

var base_url: String
var world: String
var token: String

func configure(url: String, world_id: String, access_token: String) -> bool:
 var url_pattern := RegEx.create_from_string("^http://127\\.0\\.0\\.1:[1-9][0-9]{0,4}$")
 var world_pattern := RegEx.create_from_string("^[A-Za-z0-9_-]{1,128}$")
 if url_pattern.search(url) == null or world_pattern.search(world_id) == null:
  return false
 var port := int(url.get_slice(":", 2))
 if port > 65535 or access_token.length() < 24 or access_token.length() > 512:
  return false
 if access_token.contains("\r") or access_token.contains("\n"):
  return false
 base_url = url
 world = world_id
 token = access_token
 return true

func fetch(path: String) -> Dictionary:
 return await send(HTTPClient.METHOD_GET, path)

func submit(command_id: String, command_type: String, payload: Dictionary) -> Dictionary:
 var body := JSON.stringify({"command_id": command_id, "type": command_type, "payload": payload})
 return await send(HTTPClient.METHOD_POST, "/commands", body)

func send(method: HTTPClient.Method, path: String, body: String = "") -> Dictionary:
 var request := HTTPRequest.new()
 request.timeout = 6.0
 request.max_redirects = 0
 request.body_size_limit = 65536
 add_child(request)
 var headers := PackedStringArray(["Authorization: Bearer " + token, "Content-Type: application/json"])
 var start := request.request(base_url + "/v1/worlds/" + world + path, headers, method, body)
 if start != OK:
  request.queue_free()
  return {"status": 0, "data": {}}
 var response: Array = await request.request_completed
 request.queue_free()
 if response[0] != HTTPRequest.RESULT_SUCCESS:
  return {"status": 0, "data": {}}
 var decoded: Variant = JSON.parse_string(response[3].get_string_from_utf8())
 if not decoded is Dictionary:
  return {"status": 0, "data": {}}
 return {"status": response[1], "data": decoded}
