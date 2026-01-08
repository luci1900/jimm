// Copyright 2025 Canonical.

package testing

import (
	"time"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/juju/juju/api/base"
	cloudapi "github.com/juju/juju/api/client/cloud"
	"github.com/juju/juju/api/client/modelmanager"
	"github.com/juju/juju/core/model"
	"github.com/juju/names/v5"
	gc "gopkg.in/check.v1"

	"github.com/canonical/jimm/v3/internal/testutils/jimmtest"
)

// caasModelManagerSuite requires additional setup, described in README.md#Setup microk8s cloud
type caasModelManagerSuite struct {
	jimmtest.WebsocketE2ESuite

	cred      names.CloudCredentialTag
	cloudName string
	modelUUID string
}

var _ = gc.Suite(&caasModelManagerSuite{})

func (s *caasModelManagerSuite) SetUpTest(c *gc.C) {
	s.WebsocketE2ESuite.SetUpTest(c)
	cloudclient := cloudapi.NewClient(s.Open(c, nil, "bob@canonical.com", nil))
	cloud, credential := s.GetMicrok8sCloudAndCloudCredential(c)
	s.cloudName = cloud.Name
	err := cloudclient.AddCloud(cloud, false)
	c.Assert(err, gc.Equals, nil)
	credentialName := petname.Generate(2, "-")
	s.cred = names.NewCloudCredentialTag(s.cloudName + "/bob@canonical.com/" + credentialName)
	s.UpdateCloudCredential(c, s.cred, credential)
}

func (s *caasModelManagerSuite) TearDownTest(c *gc.C) {
	if s.modelUUID != "" {
		s.DestroyModelAndDeleteFromDatabase(c, names.NewModelTag(s.modelUUID))
	}
	conn := s.Open(c, nil, "bob@canonical.com", nil)
	defer conn.Close()
	cloudclient := cloudapi.NewClient(conn)
	err := cloudclient.RemoveCloud(s.cloudName)
	c.Assert(err, gc.Equals, nil)
}

func (s *caasModelManagerSuite) TestCreateModelKubernetes(c *gc.C) {
	conn := s.Open(c, nil, "bob", nil)
	defer conn.Close()

	client := modelmanager.NewClient(conn)
	modelName := petname.Generate(2, "-")
	mi, err := client.CreateModel(modelName, "bob@canonical.com", s.cloudName, "", s.cred, nil)
	c.Assert(err, gc.Equals, nil)
	c.Assert(mi.Name, gc.Equals, modelName)
	c.Assert(mi.Type, gc.Equals, model.CAAS)
	c.Assert(mi.ProviderType, gc.Equals, "kubernetes")
	c.Assert(mi.Cloud, gc.Equals, s.cloudName)
	c.Assert(mi.CloudRegion, gc.Equals, "localhost")
	c.Assert(mi.Owner, gc.Equals, "bob@canonical.com")
	s.modelUUID = mi.UUID
}

func (s *caasModelManagerSuite) TestListCAASModelSummaries(c *gc.C) {
	conn := s.Open(c, nil, "bob", nil)
	defer conn.Close()

	client := modelmanager.NewClient(conn)
	modelName := petname.Generate(2, "-")
	mi, err := client.CreateModel(modelName, "bob@canonical.com", s.cloudName, "", s.cred, nil)
	c.Assert(err, gc.Equals, nil)
	s.modelUUID = mi.UUID

	models, err := client.ListModelSummaries("bob", false)
	c.Assert(err, gc.Equals, nil)

	var caasMS *base.UserModelSummary
	for _, m := range models {
		if m.Name == modelName {
			caasMS = &m
		}
	}
	if caasMS == nil {
		c.Fail()
	}
	expectedCaas := &base.UserModelSummary{
		Name:            modelName,
		UUID:            mi.UUID,
		Type:            "caas",
		ControllerUUID:  jimmtest.ControllerUUID,
		IsController:    false,
		ProviderType:    "kubernetes",
		DefaultSeries:   "jammy",
		Cloud:           s.cloudName,
		CloudRegion:     "localhost",
		CloudCredential: s.cloudName + "/bob@canonical.com/" + s.cred.Name(),
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
				"AgentVersion",
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
	conn := s.Open(c, nil, "bob", nil)
	defer conn.Close()

	client := modelmanager.NewClient(conn)
	modelName := petname.Generate(2, "-")
	mi, err := client.CreateModel(modelName, "bob@canonical.com", s.cloudName, "", s.cred, nil)
	c.Assert(err, gc.Equals, nil)
	s.modelUUID = mi.UUID

	models, err := client.ListModels("bob")
	c.Assert(err, gc.Equals, nil)
	c.Assert(
		models,
		jimmtest.CmpEquals(
			cmpopts.IgnoreTypes(
				&time.Time{},
			),
			cmpopts.SortSlices(func(a, b base.UserModel) bool {
				return a.Name < b.Name
			}),
		),
		[]base.UserModel{
			{
				Name:  modelName,
				UUID:  mi.UUID,
				Owner: "bob@canonical.com",
				Type:  "caas",
			}, {
				Name:  s.Model.Name,
				UUID:  s.Model.UUID.String,
				Owner: "bob@canonical.com",
				Type:  "iaas",
			},
			{
				Name:  s.Model3.Name,
				UUID:  s.Model3.UUID.String,
				Owner: "charlie@canonical.com",
				Type:  "iaas",
			},
		},
	)
}
