# List monitor groups

Returns a list of all your monitor groups. This endpoint [supports pagination](https://betterstack.com/docs/uptime/api/pagination/).

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/monitor-groups"
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
      "id": "95251342",
      "type": "monitor_group",
      "attributes": {
        "name": "Backend services",
        "sort_index": null,
        "created_at": "2020-10-03T20:20:43.547Z",
        "updated_at": "2020-10-03T20:20:43.547Z",
        "team_name": "Test team",
        "paused": true
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
  --url https://uptime.betterstack.com/api/v2/monitor-groups \
  --header "Authorization: Bearer $TOKEN"
```

# Get monitor group

Returns a single monitor group by ID.

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/monitor-groups/:monitor_group_id"
method = "GET"

[[path_param]]
name = "monitor_id"
description = "The ID of the monitor group you want to get."
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
description = '''Returns a single monitor group'''
body = '''{
  "data": {
    "id": "95251342",
    "type": "monitor_group",
    "attributes": {
      "name": "Backend services",
      "sort_index": null,
      "created_at": "2020-10-03T20:20:43.547Z",
      "updated_at": "2020-10-03T20:20:43.547Z",
      "team_name": "Test team",
      "paused": true
    }
  }
}'''

[/responses]

#### Example cURL

```shell
curl --request GET \
  --url https://uptime.betterstack.com/api/v2/monitor-groups/95251342 \
  --header "Authorization: Bearer $TOKEN"
```

# List monitors in group

Returns monitors belonging to the given monitor group. This endpoint [supports pagination](https://betterstack.com/docs/uptime/api/pagination/).

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/monitor-groups/:monitor_group_id/monitors"
method = "GET"

[[path_param]]
name = "monitor_id"
description = "The ID of the monitor group you want to get."
required = true
type = "integer"

[[header]]
name = "Authorization"
description = "Bearer `$TOKEN`"
required = true
type = "string"
[/endpoint]

[responses]
[[response]]
status = 200
description = ''''''
body = '''{
  "data": [
    {
      "id": "2",
      "type": "monitor",
      "attributes": {
        "url": "https://uptime.betterstack.com",
        "pronounceable_name": "Uptime homepage",
        "monitor_type": "keyword",
        "monitor_group_id": "12345",
        "last_checked_at": "2020-09-01T14:17:46.000Z",
        "status": "up",
        "team_name": "Test team",
        "required_keyword": "We call you",
        "verify_ssl": true,
        "check_frequency": 30,
        "call": true,
        "sms": true,
        "email": true,
        "team_wait": null,
        "http_method": "get",
        "request_timeout": 15,
        "recovery_period": 0,
        "request_body": "",
        "paused_at": null,
        "created_at": "2020-02-18T13:38:16.586Z",
        "updated_at": "2020-09-08T13:10:20.202Z",
        "ssl_expiration": 7,
        "domain_expiration": 14,
        "regions": ["us", "eu", "as", "au"],
        "port": null
      }
    }
  ],
  "pagination": {
    "first": "https://uptime.betterstack.com/api/v2/monitor-groups/95251342/monitors?page=1",
    "last": "https://uptime.betterstack.com/api/v2/monitor-groups/95251342/monitors?page=16",
    "prev": null,
    "next": "https://uptime.betterstack.com/api/v2/monitor-groups/95251342/monitors?page=2"
  }
}'''
[/responses]

#### Example cURL

```shell
curl --request GET \
  --url https://uptime.betterstack.com/api/v2/monitor-groups/95251342/monitors \
  --header "Authorization: Bearer $TOKEN"
```

# Create monitor group

Returns either a newly created monitor group, or validation errors.

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/monitor-groups"
method = "POST"

[[body_param]]
name = "team_name"
description = "Required if using global API token to specify the team which should own the resource"
required = false
type = "string"

[[body_param]]
name = "paused"
description = "Set to true to pause monitoring for any existing monitors in the group — we won't notify you about downtime. Set to false to resume monitoring for any existing monitors in the group."
required = false
type = "boolean"

[[body_param]]
name = "name"
description = "The name of the group that you can see in the dashboard."
required = false
type = "string"

[[body_param]]
name = "sort_index"
description = "Set sort_index to specify how to sort your monitor groups."
required = false
type = "integer"

[[header]]
name = "Authorization"
description = "Bearer `$TOKEN`"
required = true
type = "string"

[[header]]
name = "Content_Type"
description = "application/json"
required = false
type = "string"
[/endpoint]

[responses]
[[response]]
status = 201
description = '''Returns newly created monitor group'''
body = '''{
  "data": {
    "id": "95251342",
    "type": "monitor_group",
    "attributes": {
      "name": "Backend services",
      "sort_index": null,
      "created_at": "2020-10-03T20:20:43.547Z",
      "updated_at": "2020-10-03T20:20:43.547Z",
      "team_name": "Test team",
      "paused": true
    }
  }
}'''

[/responses]

#### Example cURL

```shell
curl --request POST \
  --url https://uptime.betterstack.com/api/v2/monitor-groups \
  --header "Authorization: Bearer $TOKEN" \
  --header 'Content-Type: application/json' \
  --data '{
	"name": "Backend services"
}'
```

# Update monitor group

Update the attributes of an existing monitor group. Send only the parameters you wish to change (e.g. `name`).

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/monitor-groups/{monitor_group_id}"
method = "PATCH"

[[path_param]]
name = "monitor_group_id"
description = "The ID of the monitor group you want to update"
required = true
type = "string"

[[body_param]]
name = "paused"
description = "Set to true to pause monitoring for any existing monitors in the group — we won't notify you about downtime. Set to false to resume monitoring for any existing monitors in the group."
required = false
type = "boolean"

[[body_param]]
name = "name"
description = "The name of the group that you can see in the dashboard."
required = false
type = "string"

[[body_param]]
name = "sort_index"
description = "Set sort_index to specify how to sort your monitor groups."
required = false
type = "integer"

[[header]]
name = "Authorization"
description = "Bearer `$TOKEN`"
required = true
type = "string"

[[header]]
name = "Content_Type"
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
      "sort_index": null,
      "created_at": "2020-10-03T20:20:43.547Z",
      "updated_at": "2020-10-03T20:20:43.547Z",
      "team_name": "Test team",
      "paused": true
    }
  }
}'''

[/responses]

#### Example cURL

```shell
curl --request PATCH \
  --url https://uptime.betterstack.com/api/v2/monitor-groups/95251342 \
  --header "Authorization: Bearer $TOKEN" \
  --header 'Content-Type: application/json' \
  --data '{
	"name": "Backend services 2"
}'
```

# Remove monitor group

Permanently deletes an existing monitor group.

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/monitor-groups/{monitor_group_id}"
method = "DELETE"

[[path_param]]
name = "monitor_group_id"
description = "The ID of the monitor group you want to delete"
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
  --url https://uptime.betterstack.com/api/v2/monitor-groups/95251342 \
  --header "Authorization: Bearer $TOKEN"
```

# Response attributes

Namespace `data`

| Parameter               | Type                         | Values                                                                                                                                                                                         |
| ----------------------- | ---------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`                    | String                       | The ID of the monitor group. String representation of number.                                                                                                                                  |
| `type`                  | String                       | `monitor_group`                                                                                                                                                                                |
| `attributes`            | Object                       | Attributes object. See below.                                                                                                                                                                  |
| `attributes.name`       | String                       | Name of the monitor group. You will see this name in the dashboard.                                                                                                                            |
| `attributes.sort_index` | Integer                      | Specifies the sorting order of the monitor groups. Can be `null`.                                                                                                                              |
| `attributes.created_at` | String (ISO DateTime format) | When was the monitor group created. Example value `"2020-10-03T20:20:43.547Z"`.                                                                                                                |
| `attributes.updated_at` | String (ISO DateTime format) | When was the monitor group last updated. Example value `"2020-10-03T20:20:43.547Z"`.                                                                                                           |
| `attributes.team_name`  | String                       | The team this monitor group is in.                                                                                                                                                             |
| `attributes.paused`     | Boolean                      | Set to `true` to pause monitoring for any existing monitors in the group — we won't notify you about downtime.<br/>Set to `false` to resume monitoring for any existing monitors in the group. |
