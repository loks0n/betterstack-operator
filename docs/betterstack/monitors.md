# List monitors

Returns list of all your monitors. This endpoint [supports pagination](https://betterstack.com/docs/uptime/api/pagination/).

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/monitors"
method = "GET"

[[query_param]]
name = "url"
description = "Filter monitors by their URL property"
required = false
type = "string"

[[query_param]]
name = "pronounceable_name"
description = "Filter monitors by their pronounceable name property"
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
description = '''Returns list of existing monitors.

<br>
The status attribute can have one of the following values:
<br>
*paused* - the monitor was paused
<br>
*pending* - the monitor was just created and it's waiting for the first check
<br>
*maintenance* - the monitor is paused because it is currently in its maintenance period
<br>
*up* - checks are passing
<br>
*validating* - service seems to be back up, but the recovery_period since the last failed check still hasn't passed
<br>
*down* - checks are failing

<br>

If an escalation policy is set (policy_id) then the simple escalation policy settings (call, sms, email, push) are ignored.'''
body = '''{
"data": [
{
"id": "2",
"type": "monitor",
"attributes": {
"url": "https://uptime.betterstack.com",
"pronounceable_name": "Uptime homepage",
"monitor_type": "keyword",
"monitor_group_id": 12345,
"last_checked_at": "2020-09-01T14:17:46.000Z",
"status": "up",
"policy_id": null,
"expiration_policy_id": null,
"team_name": "Test team",
"required_keyword": "We call you",
"verify_ssl": true,
"check_frequency": 30,
"call": true,
"sms": true,
"email": true,
"push": true,
"team_wait": null,
"http_method": "get",
"request_timeout": 15,
"recovery_period": 0,
"request_headers": [
{
"id": "123",
"name": "Content-Type",
"value": "application/xml"
}
],
"request_body": "",
"paused_at": null,
"created_at": "2020-02-18T13:38:16.586Z",
"updated_at": "2020-09-08T13:10:20.202Z",
"ssl_expiration": 7,
"domain_expiration": 14,
"regions": ["us", "eu", "as", "au"],
"maintenance_from": "01:02:00",
"maintenance_to": "03:04:00",
"maintenance_timezone": "Amsterdam",
"maintenance_days": ["mon", "tue", "wed", "thu", "fri", "sat", "sun"],
"port": null,
"confirmation_period": 120,
"expected_status_codes": [],
"environment_variables": {}
}
}
],
"pagination": {
"first": "https://uptime.betterstack.com/api/v2/monitors?page=1",
"last": "https://uptime.betterstack.com/api/v2/monitors?page=16",
"prev": null,
"next": "https://uptime.betterstack.com/api/v2/monitors?page=2"
}
}'''

[/responses]

<br>

#### Example cURL

[code-tabs]

```shell
[label List all]
curl --request GET \
  --url https://uptime.betterstack.com/api/v2/monitors \
  --header "Authorization: Bearer $TOKEN"
```

```shell
[label Filter by URL]
curl --request GET \
  --url https://uptime.betterstack.com/api/v2/monitors?url=https://google.com \
  --header "Authorization: Bearer $TOKEN"
```

[/code-tabs]

# Get monitor

Returns a single monitor.

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/monitors/{monitor_id}"
method = "GET"

[[path_param]]
name = "monitor_id"
description = "The ID of the monitor you want to get"
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
description = '''Returns a single monitor.

<br>
The status attribute can have one of the following values:
<br>
*paused* - the monitor was paused
<br>
*pending* - the monitor was just created and it's waiting for the first check
<br>
*maintenance* - the monitor is paused because it is currently in its maintenance period
<br>
*up* - checks are passing
<br>
*validating* - service seems to be back up, but the recovery_period since the last failed check still hasn't passed
<br>
*down* - checks are failing

<br>

If an escalation policy is set (policy_id) then the simple escalation policy settings (call, sms, email, push) are ignored.'''
body = '''
{
"data": {
"id": "123456789",
"type": "monitor",
"attributes": {
"url": "https://uptime.betterstack.com",
"pronounceable_name": "Uptime homepage",
"monitor_type": "keyword",
"monitor_group_id": 12345,
"last_checked_at": "2020-09-01T14:17:46.000Z",
"status": "up",
"policy_id": null,
"expiration_policy_id": null,
"team_name": "Test team",
"required_keyword": "We call you",
"verify_ssl": true,
"check_frequency": 30,
"call": true,
"sms": true,
"email": true,
"push": true,
"team_wait": null,
"http_method": "get",
"request_timeout": 15,
"recovery_period": 0,
"request_headers": [
{
"id": "123",
"name": "Content-Type",
"value": "application/xml"
}
],
"request_body": "",
"paused_at": null,
"created_at": "2020-02-18T13:38:16.586Z",
"updated_at": "2020-09-08T13:10:20.202Z",
"ssl_expiration": 7,
"domain_expiration": 14,
"regions": ["us", "eu", "as", "au"],
"maintenance_from": "01:02:00",
"maintenance_to": "03:04:00",
"maintenance_timezone": "Amsterdam",
"maintenance_days": ["mon", "tue", "wed", "thu", "fri", "sat", "sun"],
"port": null,
"confirmation_period": 120,
"expected_status_codes": [],
"environment_variables": {}
}
}
}'''
[[response]]
status = 404
description = ""
body = '''{
"errors": "Resource with provided ID was not found"
}'''
[/responses]

</br>

#### Example cURL

```shell

curl --request GET \
  --url https://uptime.betterstack.com/api/v2/monitors/123456789 \
  --header "Authorization: Bearer $TOKEN"
```

# Monitor response times

Returns the response times for a monitor (last 24h).

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/monitors/{monitor_id}/response-times"
method = "GET"

[[path_param]]
name = "monitor_id"
description = "The ID of the monitor you want to get"
required = true
type = "string"

[[header]]
name = "Authorization"
description = "Bearer `$TOKEN`"
type = "string"
required = true
[/endpoint]

[responses]
[[response]]
status = 200
description = '''The response times are grouped by the server's region.'''
body = '''{
"data": {
"id": "270985",
"type": "monitor_response_times",
"attributes": {
"regions": [
{
"region": "us",
"response_times": [
{
"at": "2025-04-03T11:00:57.000Z",
"response_time": 0.47273,
"name_lookup_time": 0.000020,
"connection_time": 0.14187,
"tls_handshake_time": 0.19929,
"data_transfer_time": 0.13154
},
{
"at": "2025-04-03T11:01:27.000Z",
"response_time": 0.41211,
"name_lookup_time": 0.000020,
"connection_time": 0.13058,
"tls_handshake_time": 0.17109,
"data_transfer_time": 0.11042
},
{
"at": "2025-04-03T11:04:27.000Z",
"response_time": 0.41943,
"name_lookup_time": 0.000020,
"connection_time": 0.10908,
"tls_handshake_time": 0.19844,
"data_transfer_time": 0.11188
}
]
},
{
"region": "eu",
"response_times": [
{
"at": "2025-04-03T11:13:27.000Z",
"response_time": 0.60383,
"name_lookup_time": 0.000020,
"connection_time": 0.15449,
"tls_handshake_time": 0.31218,
"data_transfer_time": 0.13714
},
{
"at": "2025-04-03T11:13:57.000Z",
"response_time": 0.36093,
"name_lookup_time": 0.000020,
"connection_time": 0.12147,
"tls_handshake_time": 0.12925,
"data_transfer_time": 0.11019
},
{
"at": "2025-04-03T11:17:27.000Z",
"response_time": 0.34252,
"name_lookup_time": 0.000020,
"connection_time": 0.10989,
"tls_handshake_time": 0.1224,
"data_transfer_time": 0.11021
}
]
}
]
}
}
}'''

[/responses]

#### Example cURL

```shell
curl --request GET \
  --url https://uptime.betterstack.com/api/v2/monitors/123456789/response-times \
  --header "Authorization: Bearer $TOKEN"
```

# Monitor availability

Returns availability summary for a specific monitor.

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/monitors/{monitor_id}/sla"
description = "Returns availability summary for a specific monitor"
method = "GET"

[[path_param]]
name = "monitor_id"
description = "The ID of the monitor you want to get"
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
description = '''Returns the availability summary since the monitor was created or for the specified date range.

<br>

The values of `total_downtime`, `longest_incident`, and `average_incident` are in seconds'''
body = '''{
"data": {
"id": "258338",
"type": "monitor_sla",
"attributes": {
"availability": 99.98,
"total_downtime": 600,
"number_of_incidents": 3,
"longest_incident": 300,
"average_incident": 200
}
}
}'''

[[response]]
status = 400
description = '''When the optional from and to dates are invalid (e.g., the start date is in the future or the end date is before the start date).
'''
body = '''{
"errors": "The data range is invalid. The date format could not be parsed, the start time is in the future, or the end time is after the start time."
}'''
[/responses]

#### Example cURL

```shell
curl --request GET \
  --url https://uptime.betterstack.com/api/v2/monitors/123456789/sla?from=2021-01-26&to=2021-01-27 \
  --header "Authorization: Bearer $TOKEN"
```

# Create monitor

Returns a newly created monitor or validation errors.

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/monitors"
method = "POST"

[[body_param]]
name = "team_name"
description = "Required if using global API token to specify the team which should own the resource"
required = false
type = "string"

[[body_param]]
name = "monitor_type"
description = '''
Valid values:<br/>`status` — We will check your website for a 2XX HTTP status code.<br/>`expected_status_code` — We will check if your website returned one of the values in `expected_status_codes`.<br/>`keyword` — We will check if your website contains the `required_keyword`.<br/>`keyword_absence` — We will check if your website doesn't contain the `required_keyword`.<br/>`ping` — We will ping your host specified in the `url` parameter.<br/>`tcp` — We will test a TCP port at your host specified in the url parameter (`port` is required).<br/>`udp` — We will test a UDP port at your host specified in the url parameter (`port` and `required_keyword` are required).<br/>`smtp` — We will check for a SMTP server at the host specified in the url parameter (`port` is required, and can be one of `25`, `465`, `587`, or a combination of those ports separated by a comma).<br/>`pop` — We will check for a POP3 server at the host specified in the `url` parameter (`port` is required, and can be `110`, `995`, or both).<br/>`imap` — We will check for an IMAP server at the host specified in the url parameter (`port` is required, and can be 143, 993, or both).<br/>`dns` — We will check for a DNS server at the host specified in the url parameter (`request_body` is required, and should contain the domain to query the DNS server with).<br/>`playwright` — We will run the scenario defined by `playwright_script`, identified in the UI by `scenario_name`.
'''
required = false
type = "string"

[[body_param]]
name = "url"
description = "The URL of your website or the host you want to ping. See `monitor_type` below."
required = false
type = "string"

[[body_param]]
name = "pronounceable_name"
description = "The name of the monitor"
required = false
type = "string"

[[body_param]]
name = "email"
description = "Send email alerts"
required = false
type = "boolean"

[[body_param]]
name = "sms"
description = "Send SMS alerts"
required = false
type = "boolean"

[[body_param]]
name = "call"
description = "Phone call alerts"
required = false
type = "boolean"

[[body_param]]
name = "critical_alert"
description = "Should we send a critical push notification that ignores the mute switch and Do not Disturb mode?"
required = false
type = "boolean"

[[body_param]]
name = "critical_alert"
description = "Should we send a critical alert to the on-call person?"
required = false
type = "boolean"

[[body_param]]
name = "check_frequency"
description = "Check frequency (in seconds)"
required = false
type = "integer (30)"

[[body_param]]
name = "request_headers"
description = "The request headers that will be send with the check"
required = false
type = "array of objects"

[[body_param]]
name = "expected_status_codes"
description = "An array of status codes you expect to receive from your website. These status codes are considered only if the `monitor_type` is `expected_status_code`."
required = false
type = "array of integers"

[[body_param]]
name = "domain_expiration"
description = "How many days before the domain expires do you want to be alerted? Valid values are 1, 2, 3, 7, 14, 30, and 60."
required = false
type = "integer"

[[body_param]]
name = "ssl_expiration"
description = "How many days before the SSL certificate expires do you want to be alerted? Valid values are 1, 2, 3, 7, 14, 30, and 60."
required = false
type = "integer"

[[body_param]]
name = "policy_id"
description = "Set the escalation policy for the monitor."
required = false
type = "string"

[[body_param]]
name = "expiration_policy_id"
description = "Set the expiration escalation policy for the monitor. It is used for SSL certificate and domain expiration checks. When set to `null`, an e-mail is sent to the entire team."
required = false
type = "integer"

[[body_param]]
name = "follow_redirects"
description = "Should we automatically follow redirects when sending the HTTP request?"
required = false
type = "boolean"

[[body_param]]
name = "required_keyword"
description = "Required if monitor_type is set to keyword or udp. We will create a new incident if this keyword is missing on your page."
required = false
type = "string"

[[body_param]]
name = "team_wait"
description = "How long to wait before escalating the incident alert to the team. Leave blank to disable escalating to the entire team. In seconds."
required = false
type = "integer"

[[body_param]]
name = "paused"
description = "Set to true to pause monitoring — we won't notify you about downtime. Set to false to resume monitoring."
required = false
type = "boolean"

[[body_param]]
name = "port"
description = "Required if `monitor_type` is set to `tcp`, `udp`, `smtp`, `pop`, or `imap`. `tcp` and `udp` monitors accept any ports, while smtp, pop, and imap accept only the specified ports corresponding with their servers (e.g. `25`,`465`,`587` for `smtp`)."
required = false
type = "string"

[[body_param]]
name = "regions"
description = "An array of regions to set. Allowed values are `['us', 'eu', 'as', 'au']` or any subset of these regions."
required = false
type = "array of strings"

[[body_param]]
name = "monitor_group_id"
description = "Set this attribute if you want to add this monitor to a monitor group."
required = false
type = "integer"

[[body_param]]
name = "recovery_period"
description = "How long the monitor must be up to automatically mark an incident as resolved after being down. In seconds."
required = false
type = "integer"

[[body_param]]
name = "verify_ssl"
description = "Should we verify SSL certificate validity?"
required = false
type = "boolean"

[[body_param]]
name = "confirmation_period"
description = "How long should we wait after observing a failure before we start a new incident? In seconds."
required = false
type = "integer"

[[body_param]]
name = "http_method"
description = "HTTP Method used to make a request. Valid options: GET, HEAD, POST, PUT, PATCH"
required = false
type = "string"

[[body_param]]
name = "request_timeout"
description = '''
How long to wait before timing out the request?<br/><br/>

  <ul>
    <li>• For Server and Port monitors (`ping`, `tcp`, `udp`, `smtp`, `pop`, `imap` and `dns`) timeout is specified in <b>milliseconds</b>. Valid options: 500, 1000, 2000, 3000, 5000.</li>
    <li>• For all other monitor types timeout is specified in <b>seconds</b>. Valid options: 2, 3, 5, 10, 15, 30, 45, 60.</li>
  </ul>
  When `monitor_type` is set to `playwright`, this determines the Playwright scenario timeout instead. In <b>seconds</b>. Valid options: 15, 30, 45, 60
'''
required = false
type = "integer"

[[body_param]]
name = "request_body"
description = "Request body for POST, PUT, PATCH requests. Required if `monitor_type` is set to `dns` (domain to query the DNS server with)."
required = false
type = "string"

[[body_param]]
name = "auth_username"
description = "Basic HTTP authentication username to include with the request."
required = false
type = "string"

[[body_param]]
name = "auth_password"
description = "Basic HTTP authentication password to include with the request."
required = false
type = "string"

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
name = "remember_cookies"
description = "Do you want to keep cookies when redirecting?"
required = false
type = "boolean"

[[body_param]]
name = "playwright_script"
description = "For Playwright monitors, the JavaScript source code of the scenario."
required = false
type = "string"

[[body_param]]
name = "scenario_name"
description = "For Playwright monitors, the scenario name identifying the monitor in the UI."
required = false
type = "string"

[[body_param]]
name = "environment_variables"
description = "For Playwright monitors, the environment variables that can be used in the scenario. Example: `{ \"PASSWORD\": \"passw0rd\" }`."
required = false
type = "object"

[[header]]
name = "Authorization"
description = "Bearer `$TOKEN`"
required = true
type = "string"

[[header]]
name = "Content-Type"
description = "application/json"
required = false
type = "string"
[/endpoint]

[responses]

[[response]]
status = 201
description = '''Returns newly created monitor'''
body = '''{
"data": {
"id": "238",
"type": "endpoint",
"attributes": {
"url": "https://uptime.betterstack.com",
"pronounceable_name": "Uptime homepage",
"monitor_type": "keyword",
"monitor_group_id": 12345,
"last_checked_at": "2020-09-01T14:17:46.000Z",
"status": "up",
"policy_id": null,
"expiration_policy_id": null,
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
"maintenance_days": ["mon", "tue", "wed", "thu", "fri", "sat", "sun"],
"port": null,
"expected_status_codes": [],
"environment_variables": {}
}
}
}'''

[[response]]
status = 422
description = '''Validation failed'''
body = '''{
"errors": {
"base": [
"URL is invalid."
]
}
}'''

[/responses]

#### Example cURL

```shell
curl --request POST \
  --url https://uptime.betterstack.com/api/v2/monitors \
  --header "Authorization: Bearer $TOKEN" \
  --header 'Content-Type: application/json' \
  --data '{
	"monitor_type": "status",
	"url": "https://facebook.com",
	"pronounceable_name": "Facebook homepage",
	"email": true,
	"sms": true,
	"call": true,
	"check_frequency": 30,
	"request_headers": [
	  {
	    "name": "X-Custom-Header",
	    "value": "custom header value"
	  }
	]
}'
```

# Update monitor

Update existing monitor configuration. Send only the parameters you wish to change (eg. `url`)

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/monitors/{monitor_id}"
method = "PATCH"

[[path_param]]
name = "monitor_id"
description = "The ID of the monitor you want to update"
required = true
type = "string"

[[body_param]]
name = "monitor_type"
description = '''
Valid values:<br/>`status` — We will check your website for a 2XX HTTP status code.<br/>`expected_status_code` — We will check if your website returned one of the values in `expected_status_codes`.<br/>`keyword` — We will check if your website contains the `required_keyword`.<br/>`keyword_absence` — We will check if your website doesn't contain the `required_keyword`.<br/>`ping` — We will ping your host specified in the `url` parameter.<br/>`tcp` — We will test a TCP port at your host specified in the url parameter (`port` is required).<br/>`udp` — We will test a UDP port at your host specified in the url parameter (`port` and `required_keyword` are required).<br/>`smtp` — We will check for a SMTP server at the host specified in the url parameter (`port` is required, and can be one of `25`, `465`, `587`, or a combination of those ports separated by a comma).<br/>`pop` — We will check for a POP3 server at the host specified in the `url` parameter (`port` is required, and can be `110`, `995`, or both).<br/>`imap` — We will check for an IMAP server at the host specified in the url parameter (`port` is required, and can be 143, 993, or both).<br/>`dns` — We will check for a DNS server at the host specified in the url parameter (`request_body` is required, and should contain the domain to query the DNS server with).<br/>`playwright` — We will run the scenario defined by `playwright_script`, identified in the UI by `scenario_name`.
'''
required = false
type = "string"

[[body_param]]
name = "url"
description = "The URL of your website or the host you want to ping. See `monitor_type` below."
required = false
type = "string"

[[body_param]]
name = "pronounceable_name"
description = "The name of the monitor"
required = false
type = "string"

[[body_param]]
name = "email"
description = "Send email alerts"
required = false
type = "boolean"

[[body_param]]
name = "sms"
description = "Send SMS alerts"
required = false
type = "boolean"

[[body_param]]
name = "call"
description = "Phone call alerts"
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
name = "check_frequency"
description = "Check frequency (in seconds)"
required = false
type = "integer (30)"

[[body_param]]
name = "request_headers"
description = "The request headers that will be send with the check"
required = false
type = "array of objects"

[[body_param]]
name = "expected_status_codes"
description = "An array of status codes you expect to receive from your website. These status codes are considered only if the `monitor_type` is `expected_status_code`."
required = false
type = "array of integers"

[[body_param]]
name = "domain_expiration"
description = "How many days before the domain expires do you want to be alerted? Valid values are 1, 2, 3, 7, 14, 30, and 60."
required = false
type = "integer"

[[body_param]]
name = "ssl_expiration"
description = "How many days before the SSL certificate expires do you want to be alerted? Valid values are 1, 2, 3, 7, 14, 30, and 60."
required = false
type = "integer"

[[body_param]]
name = "policy_id"
description = "Set the escalation policy for the monitor."
required = false
type = "string"

[[body_param]]
name = "expiration_policy_id"
description = "Set the expiration escalation policy for the monitor. It is used for SSL certificate and domain expiration checks. When set to `null`, an e-mail is sent to the entire team."
required = false
type = "integer"

[[body_param]]
name = "follow_redirects"
description = "Should we automatically follow redirects when sending the HTTP request?"
required = false
type = "boolean"

[[body_param]]
name = "required_keyword"
description = "Required if monitor_type is set to keyword or udp. We will create a new incident if this keyword is missing on your page."
required = false
type = "string"

[[body_param]]
name = "team_wait"
description = "How long to wait before escalating the incident alert to the team. Leave blank to disable escalating to the entire team. In seconds."
required = false
type = "integer"

[[body_param]]
name = "paused"
description = "Set to true to pause monitoring — we won't notify you about downtime. Set to false to resume monitoring."
required = false
type = "boolean"

[[body_param]]
name = "port"
description = "Required if `monitor_type` is set to `tcp`, `udp`, `smtp`, `pop`, or `imap`. `tcp` and `udp` monitors accept any ports, while smtp, pop, and imap accept only the specified ports corresponding with their servers (e.g. `25`,`465`,`587` for `smtp`)."
required = false
type = "string"

[[body_param]]
name = "ip_version"
description = "Which Internet Protocol Version should we use for our requests. Valid options are: `ipv4` - use IPv4 only, `ipv6` - use IPv6 only. When not set or set to `null`, we use both IPv4 and IPv6."
required = false
type = "string"

[[body_param]]
name = "regions"
description = "An array of regions to set. Allowed values are `['us', 'eu', 'as', 'au']` or any subset of these regions."
required = false
type = "array of strings"

[[body_param]]
name = "monitor_group_id"
description = "Set this attribute if you want to add this monitor to a monitor group."
required = false
type = "integer"

[[body_param]]
name = "recovery_period"
description = "How long the monitor must be up to automatically mark an incident as resolved after being down. In seconds."
required = false
type = "integer"

[[body_param]]
name = "verify_ssl"
description = "Should we verify SSL certificate validity?"
required = false
type = "boolean"

[[body_param]]
name = "confirmation_period"
description = "How long should we wait after observing a failure before we start a new incident? In seconds."
required = false
type = "integer"

[[body_param]]
name = "http_method"
description = "HTTP Method used to make a request. Valid options: GET, HEAD, POST, PUT, PATCH"
required = false
type = "string"

[[body_param]]
name = "request_timeout"
description = '''
How long to wait before timing out the request?<br/><br/>

  <ul>
    <li>• For Server and Port monitors (`ping`, `tcp`, `udp`, `smtp`, `pop`, `imap` and `dns`) timeout is specified in <b>milliseconds</b>. Valid options: 500, 1000, 2000, 3000, 5000.</li>
    <li>• For all other monitor types timeout is specified in <b>seconds</b>. Valid options: 2, 3, 5, 10, 15, 30, 45, 60.</li>
  </ul>
  When `monitor_type` is set to `playwright`, this determines the Playwright scenario timeout instead. In <b>seconds</b>. Valid options: 15, 30, 45, 60
'''
required = false
type = "integer"

[[body_param]]
name = "request_body"
description = "Request body for POST, PUT, PATCH requests. Required if `monitor_type` is set to `dns` (domain to query the DNS server with)."
required = false
type = "string"

[[body_param]]
name = "auth_username"
description = "Basic HTTP authentication username to include with the request."
required = false
type = "string"

[[body_param]]
name = "auth_password"
description = "Basic HTTP authentication password to include with the request."
required = false
type = "string"

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
name = "remember_cookies"
description = "Do you want to keep cookies when redirecting?"
required = false
type = "boolean"

[[body_param]]
name = "playwright_script"
description = "For Playwright monitors, the JavaScript source code of the scenario."
required = false
type = "string"

[[body_param]]
name = "scenario_name"
description = "For Playwright monitors, the scenario name identifying the monitor in the UI."
required = false
type = "string"

[[body_param]]
name = "environment_variables"
description = "For Playwright monitors, the environment variables that can be used in the scenario. Example: `{ \"PASSWORD\": \"passw0rd\" }`."
required = false
type = "object"

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
"id": "2",
"type": "monitor",
"attributes": {
"url": "https://uptime.betterstack.com",
"pronounceable_name": "Uptime homepage",
"monitor_type": "keyword",
"monitor_group_id": 12345,
"last_checked_at": "2020-09-01T14:17:46.000Z",
"status": "up",
"policy_id": null,
"expiration_policy_id": null,
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
"maintenance_from": "01:02:00",
"maintenance_to": "03:04:00",
"maintenance_timezone": "Amsterdam",
"maintenance_days": ["mon", "tue", "wed", "thu", "fri", "sat", "sun"],
"port": null,
"expected_status_codes": [],
"environment_variables": {}
}
}
}'''

[[response]]
status = 422
description = '''Validation failed'''
body = '''{
"errors": {
"base": [
"URL is invalid."
]
}
}'''
[/responses]

#### Example cURL

[code-tabs]

```shell
curl --request PATCH \
  --url https://uptime.betterstack.com/api/v2/monitors/225493 \
  --header "Authorization: Bearer $TOKEN" \
  --header 'Content-Type: application/json' \
  --data '{
	"check_frequency": 60
}'
```

```shell
[label Example with request headers]
curl --request PATCH \
  --url https://uptime.betterstack.com/api/v2/monitors/225493 \
  --header "Authorization: Bearer $TOKEN" \
  --header 'Content-Type: application/json' \
  --data '{
	"request_headers": [
	  { "name": "X-Create-New-Header", "value": "New Header Value" },
	  { "id": "123", "name": "X-Update-Existing-Header-Name" },
	  { "id": "456", "_destroy": true }
	]
}'
```

[/code-tabs]

# Remove monitor

Permanently deletes an existing monitor.

[endpoint]
base_url = "https://uptime.betterstack.com"
path = "/api/v2/monitors/{monitor_id}"
method = "DELETE"

[[path_param]]
name = "monitor_id"
description = "The ID of the monitor you want to delete"
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
  --url https://uptime.betterstack.com/api/v2/monitors/225493 \
  --header "Authorization: Bearer $TOKEN"
```

# Response attributes

Namespace `data`

| Parameter                                        | Type                         | Values                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| ------------------------------------------------ | ---------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`                                             | String                       | The id of the monitor. String representation of a number.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| `type`                                           | String                       | `monitor`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| `attributes`                                     | Object                       | Attributes object - contains all the Monitor attributes. See below.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `attributes.url`                                 | String                       | The URL of your website or the host you want to ping. See `attributes.monitor_type` below.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `attributes.pronounceable_name`                  | String                       | The pronounceable name of the monitor. We will use this when we call you, so no tongue-twisters, please. :)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| `attributes.monitor_type `                       | String                       | Valid values:<br/>`status` — We will check your website for a 2XX HTTP status code.<br/>`expected_status_code` — We will check if your website returned one of the values in `expected_status_codes`.<br/>`keyword` — We will check if your website contains the `required_keyword`.<br/>`keyword_absence` — We will check if your website doesn't contain the `required_keyword`.<br/>`ping` — We will ping your host specified in the `url` parameter.<br/>`tcp` — We will test a TCP port at your host specified in the url parameter (`port` is required).<br/>`udp` — We will test a UDP port at your host specified in the url parameter (`port` and `required_keyword` are required).<br/>`smtp` — We will check for a SMTP server at the host specified in the url parameter (`port` is required, and can be one of `25`, `465`, `587`, or a combination of those ports separated by a comma).<br/>`pop` — We will check for a POP3 server at the host specified in the `url` parameter (`port` is required, and can be `110`, `995`, or both).<br/>`imap` — We will check for an IMAP server at the host specified in the url parameter (`port` is required, and can be 143, 993, or both).<br/>`dns` — We will check for a DNS server at the host specified in the url parameter (`request_body` is required, and should contain the domain to query the DNS server with).<br/>`playwright` — We will run the scenario defined by `playwright_script`, identified in the UI by `scenario_name`. |
| `attributes.monitor_group_id`                    | Integer                      | The Id of the monitor group that your monitor is part of. Set this attribute if you want to add this monitor to a monitor group.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `attributes.last_checked_at`                     | String (ISO DateTime format) | When was the last check performed. Example value `2020-09-01T14:17:46.000Z`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `attributes.status`                              | String                       | `up` — Checks are passing.<br/>`down` — Checks are failing.<br/>`validating` — The service seems to be back up, but the `recovery_period` since the last failed check still hasn't passed.<br/>`paused` — The monitor was paused.<br/>`pending` — The monitor was just created and it's waiting for the first check.<br/>`maintenance` — The monitor is paused because it is currently in its maintenance period.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| `attributes.policy_id`                           | Integer                      | The escalation policy ID for this monitor.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `attributes.expiration_policy_id`                | Integer                      | The expiration escalation policy ID for this monitor. It is used for SSL certificate and domain expiration checks. When set to `null`, an e-mail is sent to the entire team.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `attributes.team_name`                           | String                       | The team this monitor is in.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `attributes.required_keyword`                    | String                       | Required if `monitor_type` is set to `keyword` or `udp`. We will create a new incident if this keyword is missing on your page.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `attributes.verify_ssl`                          | Boolean                      | Verify SSL certificate validity.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `attributes.check_frequency`                     | Integer                      | How often should we check your website? In seconds.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `attributes.call`                                | Boolean                      | Call the on-call person.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `attributes.sms`                                 | Boolean                      | Send an SMS to the on-call person.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `attributes.email`                               | Boolean                      | Send an email to the on-call person.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `attributes.push`                                | Boolean                      | Send a push notification to the on-call person.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `attributes.critical_alert`                      | Boolean                      | Send a critical alert push notification to the on-call person. Falls back to a regular push notification, if critical alerts are not supported.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `attributes.team_wait`                           | Integer                      | How long to wait before escalating the incident alert to the team. Leave blank to disable escalating to the entire team. In seconds.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `attributes.http_method`                         | String                       | HTTP Method used to make a request. Valid options: `GET`, `HEAD`, `POST`, `PUT`, `PATCH`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| `attributes.request_timeout`                     | Integer                      | How long to wait before timing out the request?<br/><br/><ul><li>For Server and Port monitors (`ping`, `tcp`, `udp`, `smtp`, `pop`, `imap` and `dns`) timeout is specified in <b>milliseconds</b>.</li><li>For all other monitor types timeout is specified in <b>seconds</b>.</li></ul>When `monitor_type` is set to `playwright`, this determines the Playwright scenario timeout instead. In <b>seconds</b>.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `attributes.recovery_period`                     | Integer                      | How long the monitor must be up to automatically mark an incident as resolved after being down. In seconds.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| `attributes.request_headers`                     | Array of objects             | An optional array of custom HTTP headers for the request. Set the `name` and `value` properties to form a complete header.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `attributes.request_headers[header_index].id`    | String                       | The Id of the header. Do not set this parameter; it is generated automatically.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `attributes.request_headers[header_index].name`  | String                       | The name of the header. Example value: `Application-type`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `attributes.request_headers[header_index].value` | String                       | The value of the header. Example value: `application/json`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| `attributes.request_body`                        | String                       | Request body for `POST`, `PUT`, `PATCH` requests. Required if `monitor_type` is set to `dns` (domain to query the DNS server with).                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `attributes.paused_at`                           | String (ISO DateTime format) | When was the monitor paused. `null` if the monitor is not paused.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| `attributes.created_at`                          | String (ISO DateTime format) | When was the monitor created.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `attributes.updated_at`                          | String (ISO DateTime format) | When was the monitor last updated.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `attributes.ssl_expiration`                      | Integer                      | How many days in advance of your SSL certificate expiration date you want to be alerted. "Don't check for SSL expiration" = `null` Allowed values are: `null`, `1`, `2`, `3`, `7`, `14`, `30`, `60`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `attributes.domain_expiration`                   | Integer                      | How many days in advance of your domain's expiration date you want to be alerted. "Don't check for domain expiration" = `null` Allowed values are: `null`, `1`, `2`, `3`, `7`, `14`, `30`, `60`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `attributes.regions`                             | Array of strings             | An array of regions to set. Allowed values are `["us", "eu", "as", "au"]` or any subset of these regions.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| `attributes.port`                                | String                       | Required if `monitor_type` is set to `tcp`, `udp`, `smtp`, `pop`, or `imap`. `tcp` and `udp` monitors accept any ports, while `smtp`, `pop`, and `imap` accept only the specified ports corresponding with their servers (e.g. `"25,465,587"` for `smtp`).                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `attributes.confirmation_period`                 | Integer                      | How long should we wait after observing a failure before we start a new incident? In seconds.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `attributes.expected_status_codes`               | Array of integers            | An array of status codes you expect to receive from your website. These status codes are considered only if the `monitor_type` is `expected_status_code`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| `attributes.maintenance_days`                    | Array of strings             | An array of maintenance days to set. If a maintenance window is overnight both affected days should be set. Allowed values are `['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun']` or any subset of these days.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `attributes.maintenance_from`                    | String                       | Start of the maintenance window each day. We won't check your website during this window. Example: `01:00:00`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| `attributes.maintenance_to`                      | String                       | End of the maintenance window each day. Example: `03:00:00`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `attributes.maintenance_timezone`                | String                       | The timezone to use for the maintenance window each day. Defaults to UTC. The accepted values can be found in the [Rails `TimeZone` documentation](https://api.rubyonrails.org/classes/ActiveSupport/TimeZone.html).                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `attributes.playwright_script`                   | String                       | For Playwright monitors, the JavaScript source code of the scenario.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `attributes.environment_variables`               | Object                       | For Playwright monitors, the environment variables that can be used in the scenario. Example: `{ "PASSWORD": "passw0rd" }`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
