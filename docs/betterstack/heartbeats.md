# List heartbeats

Returns a list of all your heartbeats. This endpoint [supports pagination](https://betterstack.com/docs/uptime/api/pagination/).

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/heartbeats"
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
description = '''The `status` attribute can have one of the following values:
<br> `paused` - the heartbeat was paused
<br> `pending` - the heartbeat was just created and is waiting for the first request
<br> `up` - heartbeat received request on time
<br> `down` - heartbeat did not receive a request on time'''
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
        "heartbeat_group_id": null,
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
    "first": "https://uptime.betterstack.com/api/v2/heartbeats?page=1",
    "last": "https://uptime.betterstack.com/api/v2/heartbeats?page=1",
    "prev": null,
    "next": null
  }
}'''

[/responses]

#### Example cURL

```shell
curl --request GET \
  --url https://uptime.betterstack.com/api/v2/heartbeats \
  --header "Authorization: Bearer $TOKEN"
```

# Get a single heartbeat

Returns a single heartbeat by ID.

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/heartbeats/{heartbeat_id}"
method = "GET"

[[path_param]]
name = "heartbeat_id"
description = "The ID of your requested heartbeat"
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
description = ''' The requested heartbeat.
The `status` attribute can have one of the following values:
<br> `paused` - the heartbeat was paused
<br> `pending` - the heartbeat was just created and is waiting for the first request
<br> `up` - heartbeat received request on time
<br> `down` - heartbeat did not receive a request on time'''
body = '''{
  "data": {
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
      "heartbeat_group_id": null,
      "team_name": "Test team",
      "sort_index": null,
      "paused_at": null,
      "maintenance_from": "01:02:00",
      "maintenance_to": "03:04:00",
      "maintenance_timezone": "Amsterdam",
      "maintenance_days": ["mon", "tue", "wed", "thu", "fri", "sat", "sun"],
      "created_at": "2020-05-05T11:41:49.327Z",
      "updated_at": "2020-12-10T15:00:15.089Z",
      "status": "up"
    }
  }
}'''

[/responses]

#### Example cURL

```shell
curl --request GET \
  --url https://uptime.betterstack.com/api/v2/heartbeats/12345 \
  --header "Authorization: Bearer $TOKEN"
```

# Get a heartbeat's availability summary

Returns availability summary for a specific heartbeat.

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/heartbeats/{heartbeat_id}/availability"
description = "Returns availability summary for a specific heartbeat"
method = "GET"

[[path_param]]
name = "heartbeat_id"
description = "The ID of your requested heartbeat"
required = true
type = "string"

[[query_param]]
name = "from"
description = "Start date (e.g., 2021-01-26)"
required = false
type = "string"

[[query_param]]
name = "to"
description = "End date (e.g., 2021-01-27)"
required = false
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
description = '''Returns the availability summary since the heartbeat was created or for the specified date range.

<br>

The values of `total_downtime`, `longest_incident`, and `average_incident` are in seconds'''
body = '''{
  "data": {
    "id": "824910",
    "type": "heartbeat_availability",
    "attributes": {
      "availability": 99.98,
      "total_downtime": 335,
      "number_of_incidents": 5,
      "longest_incident": 194,
      "average_incident": 67
    }
  }
}'''

[[response]]
status = 400
description = '''When the optional from and to dates are invalid (e.g., the start date is in the future or the end date is before the start date).
'''
body = '''{
  "errors": "The data range is invalid. The date format could not be parsed, the start time is in the future, or the end time is before the start time."
}'''
[/responses]

#### Example cURL

```shell
curl --request GET \
  --url https://uptime.betterstack.com/api/v2/heartbeats/123456789/availability?from=2021-01-26&to=2021-01-27 \
  --header "Authorization: Bearer $TOKEN"
```

# Create a heartbeat

Returns either a newly created heartbeat, or validation errors.

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/heartbeats"
method = "POST"

[[body_param]]
name = "team_name"
description = "Required if using global API token to specify the team which should own the resource"
required = false
type = "string"

[[body_param]]
name = "name"
description = "The name of the service for this heartbeat."
required = false
type = "string"

[[body_param]]
name = "period"
description = "How often should we expect this heartbeat? In seconds Minimum value: `30` seconds"
required = false
type = "integer"

[[body_param]]
name = "grace"
description = "Heartbeats can fluctuate; specify this value to control what is still acceptable Minimum value: `0` seconds We recommend setting this to approx. 20% of `period`"
required = false
type = "integer"

[[body_param]]
name = "call"
description = "Should we call the on-call person?"
required = false
type = "boolean"

[[body_param]]
name = "sms"
description = "Should we send an SMS to the on-call person?"
required = false
type = "boolean"

[[body_param]]
name = "email"
description = "Should we send an email to the on-call person?"
required = false
type = "boolean"

[[body_param]]
name = "push"
description = "Should we send a push notification to the on-call person?"
required = false
type = "boolean"

[[body_param]]
name = "critical_alert"
description = "Should we send a critical push notification that ignores the mute switch and Do not Disturb mode?"
required = false
type = "boolean"

[[body_param]]
name = "team_wait"
description = "How long should we wait before escalating the incident alert to the team? Leave blank to disable escalating to the entire team."
required = false
type = "integer"

[[body_param]]
name = "heartbeat_group_id"
description = "Set this attribute if you want to add this heartbeat to a heartbeat group"
required = false
type = "integer"

[[body_param]]
name = "sort_index"
description = "An index controlling the position of a heartbeat in the heartbeat group."
required = false
type = "integer"

[[body_param]]
name = "paused"
description = "Set to `true` to pause monitoring — we won't notify you about downtime. Set to `false` to resume monitoring"
required = false
type = "boolean"

[[body_param]]
name = "maintenance_days"
description = "An array of maintenance days to set. If a maintenance window is overnight both affected days should be set. Allowed values are `['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun']` or any subset of these days."
required = false
type = "array of strings"

[[body_param]]
name = "maintenance_from"
description = "Start of the maintenance window each day. We won't check your website during this window. Example: '01:00:00'"
required = false
type = "string"

[[body_param]]
name = "maintenance_to"
description = "End of the maintenance window each day. Example: '03:00:00'"
required = false
type = "string"

[[body_param]]
name = "maintenance_timezone"
description = "The timezone to use for the maintenance window each day. Defaults to UTC. The accepted values can be found in the Rails TimeZone documentation. https://api.rubyonrails.org/classes/ActiveSupport/TimeZone.html"
required = false
type = "string"

[[body_param]]
name = "policy_id"
description = "Set the escalation policy for the monitor."
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
description = '''Returns a newly created heartbeat'''
body = '''{
  "data": {
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
      "heartbeat_group_id": null,
      "team_name": "Test team",
      "sort_index": null,
      "paused_at": null,
      "maintenance_days": ["mon", "tue", "wed", "thu", "fri", "sat", "sun"],
      "created_at": "2020-05-05T11:41:49.327Z",
      "updated_at": "2020-12-10T15:00:15.089Z"
    }
  }
}'''

[[response]]
status = 422
description = '''Validation failed'''
body = '''{
  "errors": {
    "name": [
      "Can't be blank."
    ]
  }
}'''
[/responses]

#### Example cURL

```shell
curl --request POST \
  --url https://uptime.betterstack.com/api/v2/heartbeats \
  --header "Authorization: Bearer $TOKEN" \
  --header 'Content-Type: application/json' \
  --data '{
	"name": "Testing heartbeat",
	"period": 10800,
	"grace": 300
}'
```

# Update heartbeat

Update an existing heartbeat configuration. Send only the parameters you wish to change (e.g. `name`)

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/heartbeats/{heartbeat_id}"
method = "PATCH"

[[path_param]]
name = "heartbeat_id"
description = "The ID of your requested heartbeat."
required = true
type = "string"

[[body_param]]
name = "name"
description = "The name of the service for this heartbeat."
required = false
type = "string"

[[body_param]]
name = "period"
description = "How often should we expect this heartbeat? In seconds Minimum value: `30` seconds"
required = false
type = "integer"

[[body_param]]
name = "grace"
description = "Heartbeats can fluctuate; specify this value to control what is still acceptable Minimum value: `0` seconds We recommend setting this to approx. 20% of `period`"
required = false
type = "integer"

[[body_param]]
name = "call"
description = "Should we call the on-call person?"
required = false
type = "boolean"

[[body_param]]
name = "sms"
description = "Should we send an SMS to the on-call person?"
required = false
type = "boolean"

[[body_param]]
name = "email"
description = "Should we send an email to the on-call person?"
required = false
type = "boolean"

[[body_param]]
name = "push"
description = "Should we send a push notification to the on-call person?"
required = false
type = "boolean"

[[body_param]]
name = "critical_alert"
description = "Should we send a critical push notification that ignores the mute switch and Do not Disturb mode?"
required = false
type = "boolean"

[[body_param]]
name = "team_wait"
description = "How long should we wait before escalating the incident alert to the team? Leave blank to disable escalating to the entire team."
required = false
type = "integer"

[[body_param]]
name = "heartbeat_group_id"
description = "Set this attribute if you want to add this heartbeat to a heartbeat group"
required = false
type = "string"

[[body_param]]
name = "sort_index"
description = "An index controlling the position of a heartbeat in the heartbeat group."
required = false
type = "integer"

[[body_param]]
name = "maintenance_days"
description = "An array of maintenance days to set. If a maintenance window is overnight both affected days should be set. Allowed values are `['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun']` or any subset of these days."
required = false
type = "array of strings"

[[body_param]]
name = "maintenance_from"
description = "Start of the maintenance window each day. We won't check your website during this window. Example: '01:00:00'"
required = false
type = "string"

[[body_param]]
name = "maintenance_to"
description = "End of the maintenance window each day. Example: '03:00:00'"
required = false
type = "string"

[[body_param]]
name = "maintenance_timezone"
description = "The timezone to use for the maintenance window each day. Defaults to UTC. The accepted values can be found in the Rails TimeZone documentation. https://api.rubyonrails.org/classes/ActiveSupport/TimeZone.html"
required = false
type = "string"

[[body_param]]
name = "paused"
description = "Set to `true` to pause monitoring — we won't notify you about downtime. Set to `false` to resume monitoring"
required = false
type = "boolean"

[[body_param]]
name = "policy_id"
description = "Set the escalation policy for the monitor."
required = false
type = "string"

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
status = 200
description = ''''''
body = '''{
  "data": {
    "id": "12345",
    "type": "heartbeat",
    "attributes": {
      "url": "https://uptime.betterstack.com/api/v1/heartbeat/abcd1234abcd1234abcd1234",
      "name": "Testing heartbeat, with an update",
      "period": 10800,
      "grace": 300,
      "call": false,
      "sms": false,
      "email": true,
      "push": true,
      "team_wait": 180,
      "heartbeat_group_id": null,
      "team_name": "Test team",
      "sort_index": null,
      "maintenance_from": "01:02:00",
      "maintenance_to": "03:04:00",
      "maintenance_timezone": "Amsterdam",
      "maintenance_days": ["mon", "tue", "wed", "thu", "fri", "sat", "sun"],
      "paused_at": null,
      "created_at": "2020-05-05T11:41:49.327Z",
      "updated_at": "2020-12-10T15:00:15.089Z"
    }
  }
}'''

[/responses]

#### Example cURL

```shell
curl --request PATCH \
  --url https://uptime.betterstack.com/api/v2/heartbeats/12345 \
  --header "Authorization: Bearer $TOKEN" \
  --header 'Content-Type: application/json' \
  --data '{
	"name": "Testing heartbeat, with an update"
}'
```

# Remove heartbeat

Permanently deletes an existing heartbeat.

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/heartbeats/{heartbeat_id}"
method = "DELETE"

[[path_param]]
name = "heartbeat_id"
description = "The ID of your heartbeat you want to delete."
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
description = '''Returns empty response'''
body = ''''''

[/responses]

#### Example cURL

```shell
curl --request DELETE \
  --url https://uptime.betterstack.com/api/v2/heartbeats/12345 \
  --header "Authorization: Bearer $TOKEN"
```

# Response attributes

Namespace `data`

| Parameter                         | Type                         | Values                                                                                                                                                                                                                                                                                      |
|-----------------------------------|------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `id`                              | String                       | The ID of the heartbeat.                                                                                                                                                                                                                                                                    |
| `type`                            | String                       | `heartbeat`                                                                                                                                                                                                                                                                                 |
| `attributes`                      | Object                       | Attributes object. See below.                                                                                                                                                                                                                                                               |
| `attributes.url`                  | String                       | URL to your heartbeat.                                                                                                                                                                                                                                                                      |
| `attributes.name`                 | String                       | A name of the service for this heartbeat.                                                                                                                                                                                                                                                   |
| `attributes.period`               | Integer                      | How often is this heartbeat expected. In seconds. Minimum value: `30` seconds.                                                                                                                                                                                                              |
| `attributes.grace`                | Integer                      | Heartbeats can fluctuate; specify this value to control what is still acceptable. Minimum value: `0` seconds. We recommend setting this to approx. 20% of period.                                                                                                                           |
| `attributes.call`                 | Boolean                      | Call the on-call person.                                                                                                                                                                                                                                                                    |
| `attributes.sms`                  | Boolean                      | Send an SMS to the on-call person.                                                                                                                                                                                                                                                          |
| `attributes.email`                | Boolean                      | Send an email to the on-call person.                                                                                                                                                                                                                                                        |
| `attributes.push`                 | Boolean                      | Send a push notification to the on-call person.                                                                                                                                                                                                                                             |
| `attributes.critical_alert`       | Boolean                      | Send a critical alert push notification to the on-call person. Falls back to a regular push notification, if critical alerts are not supported.                                                                                                                                             |
| `attributes.team_wait`            | Integer                      | How long should we wait before escalating the incident alert to the team. Leave blank to disable escalating to the entire team.                                                                                                                                                             |
| `attributes.heartbeat_group_id`   | Integer                      | Set this attribute if you want to add this heartbeat to a heartbeat group. Can be `null`.                                                                                                                                                                                                   |
| `attributes.team_name`            | String                       | The name of the team this heartbeat is in.                                                                                                                                                                                                                                                  |
| `attributes.sort_index`           | Integer                      | An index controlling the position of a heartbeat in the heartbeat group.                                                                                                                                                                                                                    |
| `attributes.paused_at`            | String (ISO DateTime format) | When was the heartbeat paused. Can be `null` if heart beat is not paused.                                                                                                                                                                                                                   |
| `attributes.created_at`           | String (ISO DateTime format) | When was the heartbeat created. Example value `"2020-05-05T11:41:49.327Z"`.                                                                                                                                                                                                                 |
| `attributes.updated_at`           | String (ISO DateTime format) | When was the heartbeat last updated. Example value `"2020-05-05T11:41:49.327Z"`.                                                                                                                                                                                                            |
| `attributes.status`               | String                       | The status attribute can have one of the following values:<br/>`paused` - The heartbeat was paused.<br/>`pending` - The heartbeat was just created and is waiting for first request.<br/>`up` - Heartbeat received request on time.<br/>`down` - Heartbeat did not receive request on time. |
| `attributes.maintenance_days`     | Array of strings             | An array of maintenance days to set. If a maintenance window is overnight both affected days should be set. Allowed values are `['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun']` or any subset of these days.                                                                             |
| `attributes.maintenance_from`     | String                       | Start of the maintenance window each day. We won't check your website during this window. Example: `01:00:00`.                                                                                                                                                                              |
| `attributes.maintenance_to`       | String                       | End of the maintenance window each day. Example: `03:00:00`.                                                                                                                                                                                                                                |
| `attributes.maintenance_timezone` | String                       | The timezone to use for the maintenance window each day. Defaults to UTC. The accepted values can be found in the [Rails `TimeZone` documentation](https://api.rubyonrails.org/classes/ActiveSupport/TimeZone.html).                                                                        |
