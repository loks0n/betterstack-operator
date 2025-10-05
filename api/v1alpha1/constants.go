package v1alpha1

const (
	// BetterStackMonitorFinalizer handles remote monitor cleanup during deletion.
	BetterStackMonitorFinalizer = "betterstack.monitoring.loks0n/monitor-finalizer"

	// BetterStackHeartbeatFinalizer handles remote heartbeat cleanup during deletion.
	BetterStackHeartbeatFinalizer = "betterstack.monitoring.loks0n/heartbeat-finalizer"

	// BetterStackMonitorGroupFinalizer handles remote monitor group cleanup during deletion.
	BetterStackMonitorGroupFinalizer = "betterstack.monitoring.loks0n/monitorgroup-finalizer"

	// ConditionReady indicates the resource is fully reconciled.
	ConditionReady = "Ready"

	// ConditionCredentials reflects whether Better Stack API credentials are available.
	ConditionCredentials = "CredentialsAvailable"

	// ConditionSync captures the outcome of the most recent reconciliation attempt.
	ConditionSync = "Synced"
)
