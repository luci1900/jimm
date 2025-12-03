// Copyright 2025 Canonical.

package jujuapi_test

import (
	"sort"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/juju/juju/api/base"
	cloudapi "github.com/juju/juju/api/client/cloud"
	"github.com/juju/juju/api/client/modelmanager"
	"github.com/juju/juju/cloud"
	"github.com/juju/juju/core/life"
	"github.com/juju/juju/core/model"
	jujuparams "github.com/juju/juju/rpc/params"
	"github.com/juju/juju/state"
	"github.com/juju/juju/testing/factory"
	jujuversion "github.com/juju/juju/version"
	"github.com/juju/names/v5"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/canonical/jimm/v3/internal/testutils/jimmtest"
	"github.com/canonical/jimm/v3/internal/testutils/kubetest"
	"github.com/canonical/jimm/v3/internal/utils"
)

var zeroDuration = time.Duration(0)

type modelManagerStorageSuite struct {
	websocketSuite
	state   *state.PooledState
	factory *factory.Factory
}

var _ = gc.Suite(&modelManagerStorageSuite{})

func (s *modelManagerStorageSuite) SetUpTest(c *gc.C) {
	s.websocketSuite.SetUpTest(c)
	var err error
	s.state, err = s.StatePool.Get(s.Model.UUID.String)
	c.Assert(err, gc.Equals, nil)
	s.factory = factory.NewFactory(s.state.State, s.StatePool)
	s.factory.MakeUnit(c, &factory.UnitParams{
		Application: s.factory.MakeApplication(c, &factory.ApplicationParams{
			Charm: s.factory.MakeCharm(c, &factory.CharmParams{
				Name: "storage-block",
			}),
			Storage: map[string]state.StorageConstraints{
				"data": {Pool: "modelscoped"},
			},
		}),
	})
}

func (s *modelManagerStorageSuite) TearDownTest(c *gc.C) {
	s.factory = nil
	if s.state != nil {
		s.state.Release()
		s.state = nil
	}
	s.websocketSuite.TearDownTest(c)
}

func (s *modelManagerStorageSuite) TestDestroyModelWithStorageError(c *gc.C) {
	conn := s.open(c, nil, "bob")
	defer conn.Close()

	tag := s.Model.ResourceTag()
	client := modelmanager.NewClient(conn)
	err := client.DestroyModel(tag, nil, nil, nil, &zeroDuration)
	c.Assert(err, jc.Satisfies, jujuparams.IsCodeHasPersistentStorage)

	// Check the model is not now dying.
	mis, err := client.ModelInfo([]names.ModelTag{tag})
	c.Assert(err, gc.Equals, nil)
	c.Assert(mis, gc.HasLen, 1)
	c.Assert(mis[0].Error, gc.Equals, (*jujuparams.Error)(nil))
	c.Assert(mis[0].Result.Life, gc.Equals, life.Alive)
}

func (s *modelManagerStorageSuite) TestDestroyModelWithStorageDestroyStorageTrue(c *gc.C) {
	conn := s.open(c, nil, "bob")
	defer conn.Close()

	tag := s.Model.ResourceTag()
	client := modelmanager.NewClient(conn)
	err := client.DestroyModel(tag, utils.Ptr(true), nil, nil, &zeroDuration)
	c.Assert(err, gc.Equals, nil)

	// Check the model is not now dying.
	mis, err := client.ModelInfo([]names.ModelTag{tag})
	c.Assert(err, gc.Equals, nil)
	c.Assert(mis, gc.HasLen, 1)
	c.Assert(mis[0].Error, gc.Equals, (*jujuparams.Error)(nil))
	c.Assert(mis[0].Result.Life, gc.Equals, life.Dying)
}

func (s *modelManagerStorageSuite) TestDestroyModelWithStorageDestroyStorageFalse(c *gc.C) {
	conn := s.open(c, nil, "bob")
	defer conn.Close()

	tag := s.Model.ResourceTag()
	client := modelmanager.NewClient(conn)
	err := client.DestroyModel(tag, utils.Ptr(false), nil, nil, &zeroDuration)
	c.Assert(err, gc.Equals, nil)

	// Check the model is not now dying.
	mis, err := client.ModelInfo([]names.ModelTag{tag})
	c.Assert(err, gc.Equals, nil)
	c.Assert(mis, gc.HasLen, 1)
	c.Assert(mis[0].Error, gc.Equals, (*jujuparams.Error)(nil))
	c.Assert(mis[0].Result.Life, gc.Equals, life.Dying)
}

type caasModelManagerSuite struct {
	websocketSuite

	cred names.CloudCredentialTag
}

var _ = gc.Suite(&caasModelManagerSuite{})

func (s *caasModelManagerSuite) SetUpTest(c *gc.C) {
	s.websocketSuite.SetUpTest(c)

	ksrv := kubetest.NewFakeKubernetes(c)
	s.AddCleanup(func(c *gc.C) {
		ksrv.Close()
	})

	conn := s.open(c, nil, "bob")
	defer conn.Close()

	cloudclient := cloudapi.NewClient(conn)
	err := cloudclient.AddCloud(cloud.Cloud{
		Name:            "bob-cloud",
		Type:            "kubernetes",
		AuthTypes:       cloud.AuthTypes{cloud.UserPassAuthType},
		Endpoint:        ksrv.URL,
		HostCloudRegion: jimmtest.TestProviderType + "/" + jimmtest.TestCloudRegionName,
	}, false)
	c.Assert(err, gc.Equals, nil)

	s.cred = names.NewCloudCredentialTag("bob-cloud/bob@canonical.com/k8s")
	cred := cloud.NewCredential(cloud.UserPassAuthType, map[string]string{
		"username": kubetest.Username,
		"password": kubetest.Password,
	})
	res, err := cloudclient.UpdateCredentialsCheckModels(s.cred, cred)
	c.Assert(err, gc.Equals, nil)
	for _, model := range res {
		for _, err := range model.Errors {
			c.Assert(err, gc.Equals, nil)
		}
	}
}

func (s *caasModelManagerSuite) TestCreateModelKubernetes(c *gc.C) {
	conn := s.open(c, nil, "bob")
	defer conn.Close()

	client := modelmanager.NewClient(conn)
	mi, err := client.CreateModel("k8s-model-1", "bob@canonical.com", "bob-cloud", "", s.cred, nil)
	c.Assert(err, gc.Equals, nil)

	c.Assert(mi.Name, gc.Equals, "k8s-model-1")
	c.Assert(mi.Type, gc.Equals, model.CAAS)
	c.Assert(mi.ProviderType, gc.Equals, "kubernetes")
	c.Assert(mi.Cloud, gc.Equals, "bob-cloud")
	c.Assert(mi.CloudRegion, gc.Equals, "default")
	c.Assert(mi.Owner, gc.Equals, "bob@canonical.com")
}

func (s *caasModelManagerSuite) TestListCAASModelSummaries(c *gc.C) {
	conn := s.open(c, nil, "bob")
	defer conn.Close()

	client := modelmanager.NewClient(conn)
	mi, err := client.CreateModel("k8s-model-1", "bob@canonical.com", "bob-cloud", "", s.cred, nil)
	c.Assert(err, gc.Equals, nil)

	models, err := client.ListModelSummaries("bob", false)
	c.Assert(err, gc.Equals, nil)

	var caasMS *base.UserModelSummary
	for _, m := range models {
		if m.Name == "k8s-model-1" {
			caasMS = &m
		}
	}
	if caasMS == nil {
		c.Fail()
	}
	expectedCaas := &base.UserModelSummary{
		Name:            "k8s-model-1",
		UUID:            mi.UUID,
		Type:            "caas",
		ControllerUUID:  jimmtest.ControllerUUID,
		IsController:    false,
		ProviderType:    "kubernetes",
		DefaultSeries:   "jammy",
		Cloud:           "bob-cloud",
		CloudRegion:     "default",
		CloudCredential: "bob-cloud/bob@canonical.com/k8s",
		Owner:           "bob@canonical.com",
		Life:            "alive",
		Status: base.Status{
			Status: "available",
			Info:   "",
			Data:   map[string]interface{}{},
			Since:  nil,
		},
		ModelUserAccess:    "admin",
		UserLastConnection: nil,
		Counts:             []base.EntityCount{},
		AgentVersion:       &jujuversion.Current,
		Error:              nil,
		Migration:          nil,
		SLA: &base.SLASummary{
			Level: "",
			Owner: "bob@canonical.com",
		},
	}
	c.Assert(
		caasMS,
		jimmtest.CmpEquals(
			cmpopts.IgnoreFields(
				base.UserModelSummary{},
				"DefaultSeries",
			),
			cmpopts.IgnoreTypes(
				&time.Time{},
				&base.MigrationSummary{},
			),
		),
		expectedCaas,
	)
}

func (s *caasModelManagerSuite) TestListCAASModels(c *gc.C) {
	conn := s.open(c, nil, "bob")
	defer conn.Close()

	client := modelmanager.NewClient(conn)
	mi, err := client.CreateModel("k8s-model-1", "bob@canonical.com", "bob-cloud", "", s.cred, nil)
	c.Assert(err, gc.Equals, nil)

	models, err := client.ListModels("bob")
	c.Assert(err, gc.Equals, nil)
	sort.Slice(models, func(i, j int) bool { return i < j })

	c.Assert(
		models,
		jimmtest.CmpEquals(
			cmpopts.IgnoreTypes(
				&time.Time{},
			),
		),
		[]base.UserModel{
			{
				Name:  "k8s-model-1",
				UUID:  mi.UUID,
				Owner: "bob@canonical.com",
				Type:  "caas",
			}, {
				Name:  "model-1",
				UUID:  s.Model.UUID.String,
				Owner: "bob@canonical.com",
				Type:  "iaas",
			}, {
				Name:  "model-3",
				UUID:  s.Model3.UUID.String,
				Owner: "charlie@canonical.com",
				Type:  "iaas",
			},
		},
	)
}
