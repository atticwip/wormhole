package common

import "github.com/certusone/wormhole/bridge/pkg/readiness"

const (
	ReadinessEthSyncing    readiness.Component = "ethSyncing"
	ReadinessSolanaSyncing readiness.Component = "solanaSyncing"
	ReadinessTerraSyncing  readiness.Component = "terraSyncing"
	ReadinessQtumSyncing    readiness.Component = "qtumSyncing"
)
