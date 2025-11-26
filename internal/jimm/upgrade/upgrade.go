// Copyright 2025 Canonical.

// upgrade package provides functionality to manage the upgrade process
// for controllers in JIMM.
package upgrade

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/juju/zaputil/zapctx"
	"go.uber.org/zap"

	"github.com/canonical/jimm/v3/internal/errors"
	"github.com/canonical/jimm/v3/internal/jimm/bootstrap"
	"github.com/canonical/jimm/v3/internal/openfga"
)

// BootstrapManager defines the bootstrap manager methods required by the upgrade manager.
type BootstrapManager interface {
	WaitForJobCompletion(ctx context.Context, jobId uuid.UUID, config bootstrap.WaitConfig) error
	StartBootstrapJob(ctx context.Context, user *openfga.User, params bootstrap.BootstrapParams) (string, error)
}

// upgradeManager provides a means to manage controller upgrades within JIMM.
type upgradeManager struct {
	bootstrapManager BootstrapManager
}

// NewUpgradeManager creates a new UpgradeManager instance.
func NewUpgradeManager(
	bootstrapManager BootstrapManager,
) (*upgradeManager, error) {
	if bootstrapManager == nil {
		return nil, errors.E("bootstrap manager cannot be nil")
	}
	return &upgradeManager{
		bootstrapManager: bootstrapManager,
	}, nil
}

// UpgradeTo upgrades a controller by fetching its configuration and initiating
// a bootstrap job with that configuration, then waits for the bootstrap to complete.
func (u *upgradeManager) UpgradeTo(ctx context.Context, user *openfga.User, params UpgradeParams) error {
	if user == nil {
		return errors.E("user cannot be nil")
	}

	zapctx.Info(ctx, "starting controller upgrade", zap.String("controller-name", params.ControllerName))

	// Start the bootstrap job
	jobId, err := u.bootstrapManager.StartBootstrapJob(ctx, user, bootstrap.BootstrapParams{
		CLIVersion:         params.CLIVersion,
		CloudNameAndRegion: params.CloudNameAndRegion,
		ControllerName:     params.ControllerName,
		CloudCred:          params.CloudCred,
		PersonalCloud:      params.PersonalCloud,
		UserConfig:         params.UserConfig,
	})
	if err != nil {
		return errors.E(fmt.Errorf("failed to start bootstrap job: %w", err))
	}
	parsedJobId, err := uuid.Parse(jobId)
	if err != nil {
		return errors.E(fmt.Errorf("failed to parse bootstrap job ID: %w", err))
	}
	// Wait for the bootstrap job to complete
	if err := u.bootstrapManager.WaitForJobCompletion(ctx, parsedJobId, bootstrap.WaitConfig{}); err != nil {
		return errors.E(fmt.Errorf("bootstrap job failed: %w", err))
	}

	return nil
}
