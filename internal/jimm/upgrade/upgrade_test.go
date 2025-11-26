// Copyright 2025 Canonical.

package upgrade_test

import (
	"errors"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/frankban/quicktest/qtsuite"
	"go.uber.org/mock/gomock"

	"github.com/canonical/jimm/v3/internal/jimm/upgrade"
	"github.com/canonical/jimm/v3/internal/jimm/upgrade/mocks"
	"github.com/canonical/jimm/v3/internal/openfga"
)

type upgradeManagerSuite struct {
	bootstrapManager *mocks.MockBootstrapManager
}

func (s *upgradeManagerSuite) setupTest(c *qt.C) *gomock.Controller {
	ctrl := gomock.NewController(c)

	s.bootstrapManager = mocks.NewMockBootstrapManager(ctrl)
	return ctrl
}

func (s *upgradeManagerSuite) TestNewUpgradeManager(c *qt.C) {
	defer s.setupTest(c)

	_, err := upgrade.NewUpgradeManager(s.bootstrapManager)
	c.Assert(err, qt.IsNil)
}

func (s *upgradeManagerSuite) TestNewUpgradeManager_InvalidParams(c *qt.C) {
	ctrl := s.setupTest(c)
	defer ctrl.Finish()

	_, err := upgrade.NewUpgradeManager(nil)
	c.Assert(err, qt.ErrorMatches, "bootstrap manager cannot be nil")
}

func (s *upgradeManagerSuite) TestUpgradeTo_Success(c *qt.C) {
	ctrl := s.setupTest(c)
	defer ctrl.Finish()

	upgradeMgr, err := upgrade.NewUpgradeManager(s.bootstrapManager)
	c.Assert(err, qt.IsNil)

	jobId := "550e8400-e29b-41d4-a716-446655440000"

	s.bootstrapManager.EXPECT().
		StartBootstrapJob(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(jobId, nil)

	s.bootstrapManager.EXPECT().
		WaitForJobCompletion(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	err = upgradeMgr.UpgradeTo(c.Context(), &openfga.User{}, upgrade.UpgradeParams{})
	c.Assert(err, qt.IsNil)
}

func (s *upgradeManagerSuite) TestUpgradeTo_Error(c *qt.C) {
	ctrl := s.setupTest(c)
	defer ctrl.Finish()

	upgradeMgr, err := upgrade.NewUpgradeManager(s.bootstrapManager)
	c.Assert(err, qt.IsNil)

	errorToReturn := errors.New("bootstrap error")
	s.bootstrapManager.EXPECT().
		StartBootstrapJob(gomock.Any(), gomock.Any(), gomock.Any()).
		Return("", errorToReturn)

	err = upgradeMgr.UpgradeTo(c.Context(), &openfga.User{}, upgrade.UpgradeParams{})
	c.Assert(err, qt.ErrorMatches, ".*failed to start bootstrap job.*bootstrap error.*")
}

func (s *upgradeManagerSuite) TestUpgradeTo_WaitForJobCompletionError(c *qt.C) {
	ctrl := s.setupTest(c)
	defer ctrl.Finish()

	upgradeMgr, err := upgrade.NewUpgradeManager(s.bootstrapManager)
	c.Assert(err, qt.IsNil)

	jobId := "550e8400-e29b-41d4-a716-446655440000"
	errorToReturn := errors.New("job failed")

	s.bootstrapManager.EXPECT().
		StartBootstrapJob(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(jobId, nil)

	s.bootstrapManager.EXPECT().
		WaitForJobCompletion(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errorToReturn)

	err = upgradeMgr.UpgradeTo(c.Context(), &openfga.User{}, upgrade.UpgradeParams{})
	c.Assert(err, qt.ErrorMatches, ".*bootstrap job failed.*job failed.*")
}

//go:generate mockgen -typed -destination=./mocks/bootstrapmanager.go -package=mocks . BootstrapManager
func TestUpgradeManager(t *testing.T) {
	qtsuite.Run(qt.New(t), &upgradeManagerSuite{})
}
