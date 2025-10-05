# List heartbeat groups

Returns a list of all your heartbeat groups. This endpoint [supports pagination](https://betterstack.com/docs/uptime/api/pagination/).

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/heartbeat-groups"
method = "GET"

[[header]]
name = "Authorization"
description = "Bearer `$TOKEN`"
required = true
type = "string"
[/endpoint]

[responses]
[[response]]
status = 200
description = '''Returns list of existing monitor groups'''
body = '''{
  "data": [
    {
      "id": "123456",
      "type": "hearbeat_group",
      "attributes": {
        "name": "Backend services",
        "sort_index": 0,
        "created_at": "2020-10-03T20:20:43.547Z",
        "updated_at": "2020-10-03T20:20:43.547Z",
        "team_name": "Test team",
        "paused": false
      }
    }
  ],
  "pagination": {
    "first": "https://uptime.betterstack.com/api/v2/monitor-groups?page=1",
    "last": "https://uptime.betterstack.com/api/v2/monitor-groups?page=1",
    "prev": null,
    "next": null
  }
}'''
[/responses]

#### Example cURL

```shell
curl --request GET \
  --url https://uptime.betterstack.com/api/v2/heartbeat-groups \
  --header "Authorization: Bearer $TOKEN"
```

# Get heartbeat group

Returns an existing heartbeat group by ID.

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/heartbeat-groups/{heartbeat_group_id}"
method = "GET"

[[query_param]]
name = "heartbeat_group_id"
description = "The ID of your heartbeat group."
required = true
type = "string"

[[header]]
name = "Authorization"
description = "Bearer `$TOKEN`"
required = true
type = "string"
[/endpoint]

[responses]
[[response]]
status = 200
description = '''Returns an existing heartbeat group'''
body = '''{
  "data": {
    "id": "123456",
    "type": "heartbeat_group",
    "attributes": {
      "name": "Backend services",
      "sort_index": 0,
      "created_at": "2020-09-18T17:20:42.514Z",
      "updated_at": "2020-09-18T17:21:27.251Z",
      "team_name": "Test team",
      "paused": false
    }
  }
}'''
[/responses]

#### Example cURL

```shell
curl --request GET \
  --url https://uptime.betterstack.com/api/v2/heartbeat-groups/123456 \
  --header "Authorization: Bearer $TOKEN"
```

# List heartbeats in a group

Returns heartbeats belonging to the given heartbeat group. This endpoint [supports pagination](https://betterstack.com/docs/uptime/api/pagination/).

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/heartbeat-groups/{heartbeat_group_id}/heartbeats"
method = "GET"

[[path_param]]
name = "heartbeat_group_id"
description = "The ID of the heartbeat group you want to get heartbeats from."
required = true
type = "string"

[[header]]
name = "Authorization"
description = "Bearer `$TOKEN`"
required = true
type = "string"
[/endpoint]

[responses]
[[response]]
status = 200
description = '''Returns a paginated list of heartbeats in the specified group.'''
body = '''{
  "data": [
    {
      "id": "12345",
      "type": "heartbeat",
      "attributes": {
        "url": "https://uptime.betterstack.com/api/v1/heartbeat/abcd1234abcd1234abcd1234",
        "name": "Testing heartbeat",
        "period": 10800,
        "grace": 300,
        "call": false,
        "sms": false,
        "email": true,
        "push": true,
        "team_wait": 180,
        "heartbeat_group_id": 9525,
        "team_name": "Test team",
        "sort_index": null,
        "paused_at": null,
        "maintenance_from": "01:02:00",
        "maintenance_to": "03:04:00",
        "maintenance_timezone": "Amsterdam",
        "maintenance_days": ["mon", "tue", "wed", "thu", "fri", "sat", "sun"],
        "created_at": "2020-05-05T11:41:49.327Z",
        "updated_at": "2020-12-10T14:40:23.436Z",
        "status": "up"
      }
    }
  ],
  "pagination": {
    "first": "https://uptime.betterstack.com/api/v2/heartbeat-groups/9525/heartbeats?page=1",
    "last": "https://uptime.betterstack.com/api/v2/heartbeat-groups/9525/heartbeats?page=1",
    "prev": null,
    "next": null
  }
}'''
[/responses]

#### Example cURL

```shell
curl --request GET \
  --url https://uptime.betterstack.com/api/v2/heartbeat-groups/9525/heartbeats \
  --header "Authorization: Bearer $TOKEN"
```

# Create heartbeat group

Returns either a newly created heartbeat group, or validation errors.

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/heartbeat-groups"
method = "POST"

[[body_param]]
name = "team_name"
description = "Required if using global API token to specify the team which should own the resource"
required = false
type = "string"

[[body_param]]
name = "paused"
description = "Set to `true` to pause monitoring for any existing heartbeats in the group — we won't notify you about downtime.  Set to `false` to resume monitoring for any existing heartbeats in the group."
required = false
type = "boolean"

[[body_param]]
name = "name"
description = "A name of the group that you can see in the dashboard."
required = false
type = "string"

[[body_param]]
name = "sort_index"
description = "Set `sort_index` to specify how to sort your heartbeat groups."
required = false
type = "integer"

[[header]]
name = "Authorization"
description = "Bearer `$TOKEN`"
required = true
type = "string"

[[header]]
name = "Content-type"
description = "application/json"
required = false
type = "String"
[/endpoint]

[responses]
[[response]]
status = 201
description = '''Returns newly created heartbeat group'''
body = '''{
  "data": {
    "id": "123456",
    "type": "heartbeat_group",
    "attributes": {
      "name": "Backend services",
      "sort_index": 0,
      "created_at": "2020-10-03T20:20:43.547Z",
      "updated_at": "2020-10-03T20:20:43.547Z",
      "team_name": "Test team",
      "paused": false
    }
  }
}'''
[/responses]

#### Example cURL

```shell
curl --request POST \
  --url https://uptime.betterstack.com/api/v2/heartbeat-groups \
  --header "Authorization: Bearer $TOKEN" \
  --header 'Content-Type: application/json' \
  --data '{
	"name": "Backend services"
}'
```

# Update heartbeat group

Update the attributes of an existing heartbeat group. Send only the parameters you wish to change (e.g. `name`)

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/heartbeat-groups/{heartbeat_group_id}"
method = "PATCH"

[[path_param]]
name = "heartbeat_group_id"
description = "The ID of the heartbeat group you want to update"
required = true
type = "string"

[[body_param]]
name = "period"
description = "How often should we expect this heartbeat? In seconds Minimum value: `30` seconds"
required = false
type = "integer"

[[body_param]]
name = "paused"
description = "Set to `true` to pause monitoring for any existing heartbeats in the group — we won't notify you about downtime.  Set to `false` to resume monitoring for any existing heartbeats in the group."
required = false
type = "boolean"

[[body_param]]
name = "name"
description = "A name of the group that you can see in the dashboard."
required = false
type = "string"

[[body_param]]
name = "sort_index"
description = "Set `sort_index` to specify how to sort your heartbeat groups."
required = false
type = "integer"

[[header]]
name = "Authorization"
description = "Bearer `$TOKEN`"
required = true
type = "string"

[[header]]
name = "Content-type"
description = "application/json"
required = false
type = "string"
[/endpoint]

[responses]
[[response]]
status = 200
description = '''Returns updated monitor data'''
body = '''{
  "data": {
    "id": "95251342",
    "type": "monitor_group",
    "attributes": {
      "name": "Backend services 2",
      "sort_index": 0,
      "created_at": "2020-10-03T20:20:43.547Z",
      "updated_at": "2020-10-03T20:20:43.547Z",
      "team_name": "Test team",
      "paused": false
    }
  }
}'''
[/responses]

#### Example cURL - Change name only

```shell
curl --request PATCH \
  --url https://uptime.betterstack.com/api/v2/heartbeat-groups/95251342 \
  --header "Authorization: Bearer $TOKEN" \
  --header 'Content-Type: application/json' \
  --data '{
	"name": "Backend services 2"
}'
```

# Remove heartbeat group

Permanently deletes an existing heartbeat group.

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/heartbeat-groups/{heartbeat_group_id}"
method = "DELETE"

[[path_param]]
name = "heartbeat_group_id"
description = "The ID of the heartbeat group you want to delete."
required = true
type = "string"

[[header]]
name = "Authorization"
description = "Bearer `$TOKEN`"
required = true
type = "string"
[/endpoint]

[responses]
[[response]]
status = 204
description = '''Returns empty body'''
body = ''''''
[/responses]

#### Example cURL
```shell
curl --request DELETE \
  --url https://uptime.betterstack.com/api/v2/heartbeat-groups/95251342 \
  --header "Authorization: Bearer $TOKEN"
```

# Response attributes

Namespace `data`

| Parameter               | Type                         | Values                                                                                     |
| ----------------------- | ---------------------------- | ------------------------------------------------------------------------------------------ |
| `id`                    | String                       | The ID of the heartbeat group.                                                             |
| `type`                  | String                       | `heartbeat_group`                                                                          |
| `attributes`            | Object                       | Attributes object. See below.                                                              |
| `attributes.name`       | String                       | A name of the group that you can see in the dashboard.                                     |
| `attributes.sort_index` | Integer                      | Set `sort_index` to specify how to sort your heartbeat groups.                             |
| `attributes.created_at` | String (ISO DateTime format) | When was the group created. Example value `"2020-09-18T17:20:42.514Z"`.                    |
| `attributes.updated_at` | String (ISO DateTime format) | When was the group last updated. Example value `"2020-09-18T17:20:42.514Z"`.               |
| `attributes.team_name`  | String                       | The team this heartbeat group is in.                                                       |
| `attributes.paused`     | Boolean                      | Whether is the group paused. All heartbeats in the paused heartbeat group are also paused. |
