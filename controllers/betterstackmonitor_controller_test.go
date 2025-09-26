package controllers

import (
	"reflect"
	"testing"

	monitoringv1alpha1 "loks0n/betterstack-operator/api/v1alpha1"
)

func TestBuildMonitorAttributes(t *testing.T) {
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
		"port":                  spec.Port,
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

	got := buildMonitorAttributes(spec)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected attributes map: diff=%v", diffMaps(got, want))
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
