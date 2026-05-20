// Copyright 2025 Canonical.

package jujuclient

import (
	"github.com/juju/juju/api/client/modelupgrader"
	"github.com/juju/version/v2"
)

// AbortModelUpgrade aborts and archives any in-progress upgrade on the given model.
func (c Connection) AbortModelUpgrade(modelUUID string) error {
	return modelupgrader.NewClient(&c).AbortModelUpgrade(modelUUID)
}

// UpgradeModel upgrades the model to the provided agent version.
// The provided target version could be version.Zero, in which case the
// best version is selected by the controller and returned as ChosenVersion
// in the result.
func (c Connection) UpgradeModel(
	modelUUID string,
	targetVersion version.Number,
	stream string,
	ignoreAgentVersions bool,
	dryRun bool,
) (version.Number, error) {
	return modelupgrader.NewClient(&c).UpgradeModel(modelUUID, targetVersion, stream, ignoreAgentVersions, dryRun)
}
