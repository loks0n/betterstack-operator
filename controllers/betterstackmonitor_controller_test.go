package controllers

import (
	"encoding/json"
	"reflect"
	"testing"

	monitoringv1alpha1 "loks0n/betterstack-operator/api/v1alpha1"
	"loks0n/betterstack-operator/pkg/betterstack"
)

func TestBuildMonitorRequest(t *testing.T) {
	spec := monitoringv1alpha1.BetterStackMonitorSpec{
		URL:                       "https://example.com",
		Name:                      "Example",
		MonitorType:               "status",
		TeamName:                  "SRE",
		CheckFrequencyMinutes:     3,
		Regions:                   []string{"us", "eu"},
		RequestMethod:             "POST",
		ExpectedStatusCodes:       []int{201, 202},
		RequiredKeyword:           "healthy",
		Paused:                    true,
		Email:                     boolPtr(false),
		SMS:                       boolPtr(true),
		Call:                      boolPtr(false),
		Push:                      boolPtr(true),
		CriticalAlert:             boolPtr(true),
		FollowRedirects:           boolPtr(true),
		VerifySSL:                 boolPtr(false),
		RememberCookies:           boolPtr(true),
		PolicyID:                  "policy-1",
		ExpirationPolicyID:        "exp-1",
		MonitorGroupID:            "group-1",
		TeamWaitSeconds:           120,
		DomainExpirationDays:      14,
		SSLExpirationDays:         30,
		Port:                      443,
		RequestTimeoutSeconds:     30,
		RecoveryPeriodSeconds:     300,
		ConfirmationPeriodSeconds: 60,
		IPVersion:                 "ipv6",
		MaintenanceDays:           []string{"mon", "tue"},
		MaintenanceFrom:           "01:00:00",
		MaintenanceTo:             "02:00:00",
		MaintenanceTimezone:       "UTC",
		RequestHeaders: []monitoringv1alpha1.BetterStackHeader{{
			Name:  "Content-Type",
			Value: "application/json",
		}},
		RequestBody:          "{}",
		AuthUsername:         "user",
		AuthPassword:         "pass",
		EnvironmentVariables: map[string]string{"TOKEN": "value"},
		PlaywrightScript:     "console.log('ok')",
		ScenarioName:         "Scenario",
		AdditionalAttributes: map[string]string{"custom": "value"},
	}

	want := map[string]any{
		"url":                   spec.URL,
		"pronounceable_name":    spec.Name,
		"monitor_type":          spec.MonitorType,
		"team_name":             spec.TeamName,
		"check_frequency":       spec.CheckFrequencyMinutes * 60,
		"regions":               spec.Regions,
		"http_method":           "post",
		"expected_status_codes": spec.ExpectedStatusCodes,
		"required_keyword":      spec.RequiredKeyword,
		"paused":                true,
		"email":                 false,
		"sms":                   true,
		"call":                  false,
		"push":                  true,
		"critical_alert":        true,
		"follow_redirects":      true,
		"verify_ssl":            false,
		"remember_cookies":      true,
		"policy_id":             spec.PolicyID,
		"expiration_policy_id":  spec.ExpirationPolicyID,
		"monitor_group_id":      spec.MonitorGroupID,
		"team_wait":             spec.TeamWaitSeconds,
		"domain_expiration":     spec.DomainExpirationDays,
		"ssl_expiration":        spec.SSLExpirationDays,
		"port":                  "443",
		"request_timeout":       spec.RequestTimeoutSeconds,
		"recovery_period":       spec.RecoveryPeriodSeconds,
		"confirmation_period":   spec.ConfirmationPeriodSeconds,
		"ip_version":            spec.IPVersion,
		"maintenance_days":      spec.MaintenanceDays,
		"maintenance_from":      spec.MaintenanceFrom,
		"maintenance_to":        spec.MaintenanceTo,
		"maintenance_timezone":  spec.MaintenanceTimezone,
		"request_headers":       []map[string]string{{"name": "Content-Type", "value": "application/json"}},
		"request_body":          spec.RequestBody,
		"auth_username":         spec.AuthUsername,
		"auth_password":         spec.AuthPassword,
		"environment_variables": spec.EnvironmentVariables,
		"playwright_script":     spec.PlaywrightScript,
		"scenario_name":         spec.ScenarioName,
		"custom":                "value",
	}
	wantedJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal want: %v", err)
	}
	wanted := map[string]any{}
	if err := json.Unmarshal(wantedJSON, &wanted); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}

	gotReq := buildMonitorRequest(spec, nil)
	encoded, err := json.Marshal(gotReq)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	got := map[string]any{}
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if diff := diffMaps(got, wanted); len(diff) > 0 {
		t.Fatalf("unexpected attributes map: diff=%v", diff)
	}
}

func TestBuildMonitorRequestConvertsTimeoutForServerMonitors(t *testing.T) {
	spec := monitoringv1alpha1.BetterStackMonitorSpec{
		URL:                   "tcp://example.com",
		MonitorType:           "tcp",
		RequestTimeoutSeconds: 3,
	}

	req := buildMonitorRequest(spec, nil)
	if req.RequestTimeout == nil {
		t.Fatalf("request timeout missing")
	}
	if got, want := *req.RequestTimeout, 3000; got != want {
		t.Fatalf("timeout not converted, got %d want %d", got, want)
	}
}

func TestBuildMonitorRequestAssignsHeaderIDsWhenPresent(t *testing.T) {
	existingHeaderID := "hdr-123"
	existing := &betterstack.Monitor{
		Attributes: betterstack.MonitorAttributes{
			RequestHeaders: []betterstack.MonitorHeader{{
				ID:    existingHeaderID,
				Name:  "X-Test",
				Value: "old",
			}},
		},
	}

	spec := monitoringv1alpha1.BetterStackMonitorSpec{
		URL: "https://example.com",
		RequestHeaders: []monitoringv1alpha1.BetterStackHeader{{
			Name:  "X-Test",
			Value: "new",
		}},
	}

	req := buildMonitorRequest(spec, existing)
	if len(req.RequestHeaders) != 1 {
		t.Fatalf("expected 1 header, got %d", len(req.RequestHeaders))
	}
	if req.RequestHeaders[0].ID == nil || *req.RequestHeaders[0].ID != existingHeaderID {
		t.Fatalf("expected header id %s, got %v", existingHeaderID, req.RequestHeaders[0].ID)
	}
}

func diffMaps(got, want map[string]any) map[string][2]any {
	diff := make(map[string][2]any)
	keys := make(map[string]struct{})
	for k := range got {
		keys[k] = struct{}{}
	}
	for k := range want {
		keys[k] = struct{}{}
	}
	for k := range keys {
		gv, gok := got[k]
		wv, wok := want[k]
		if !gok || !wok || !reflect.DeepEqual(gv, wv) {
			diff[k] = [2]any{gv, wv}
		}
	}
	return diff
}

func boolPtr(v bool) *bool {
	return &v
}
