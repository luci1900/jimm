// Copyright 2025 Canonical.

package jimmtest

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/juju/juju/api"
	"github.com/juju/juju/cloud"
	jclient "github.com/juju/juju/jujuclient"
	"github.com/juju/juju/rpc/jsoncodec"
	jujuparams "github.com/juju/juju/rpc/params"
	"github.com/juju/names/v5"
	gc "gopkg.in/check.v1"
	"gopkg.in/yaml.v3"

	"github.com/canonical/jimm/v3/internal/dbmodel"
	"github.com/canonical/jimm/v3/internal/jimmhttp"
	"github.com/canonical/jimm/v3/internal/jujuapi"
)

const (
	// ControllersConfigEnvVar is the environment variable that specifies
	// the path to the controllers config file.
	ControllersConfigEnvVar = "JIMM_BACKING_CONTROLLER_CONFIG"
	TestE2EProviderType     = "lxd"
	TestE2ECloudName        = "localhost"
)

// ControllerInfo holds the controller connection information
// retrieved from the configuration file.
type ControllerInfo struct {
	UUID     string `yaml:"uuid"`
	Addrs    string `yaml:"addrs"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	CACert   string `yaml:"ca-cert"`
}

// ControllersConfig holds the top-level structure of the controller.yaml file.
type ControllersConfig struct {
	Controllers map[string]ControllerInfo `yaml:"controllers"`
}

// GetControllersConfig reads and parses the controller.yaml configuration file
// from the path specified in the JIMM_BACKING_CONTROLLER_CONFIG environment variable.
func GetControllersConfig(c *gc.C) *ControllersConfig {
	configPath := os.Getenv(ControllersConfigEnvVar)
	c.Assert(configPath, gc.Not(gc.Equals), "", gc.Commentf(
		"%s environment variable is not set. "+
			"Set it to the path of your controllers.yaml file or configure it in VS Code settings.",
		ControllersConfigEnvVar))

	data, err := os.ReadFile(configPath)
	c.Assert(err, gc.IsNil, gc.Commentf(
		"failed to read controller config file: %s. "+
			"Generate it using 'make generate-test-env'", configPath))

	var config ControllersConfig
	err = yaml.Unmarshal(data, &config)
	c.Assert(err, gc.IsNil, gc.Commentf("failed to parse controller config file"))

	c.Assert(len(config.Controllers), gc.Not(gc.Equals), 0, gc.Commentf("no controllers found in config"))

	return &config
}

// Validate asserts that all required fields are set for this controller.
func (info *ControllerInfo) Validate(c *gc.C, name string) {
	c.Assert(info.UUID, gc.Not(gc.Equals), "", gc.Commentf("uuid is not set for controller %q", name))
	c.Assert(info.Addrs, gc.Not(gc.Equals), "", gc.Commentf("addrs is not set for controller %q", name))
	c.Assert(info.Username, gc.Not(gc.Equals), "", gc.Commentf("username is not set for controller %q", name))
	c.Assert(info.Password, gc.Not(gc.Equals), "", gc.Commentf("password is not set for controller %q", name))
	c.Assert(info.CACert, gc.Not(gc.Equals), "", gc.Commentf("ca-cert is not set for controller %q", name))
}

// ToAPIInfo converts the ControllerInfo to a juju api.Info struct.
func (info *ControllerInfo) ToAPIInfo() *api.Info {
	return &api.Info{
		ControllerUUID: info.UUID,
		Addrs:          []string{info.Addrs},
		Tag:            names.NewUserTag(info.Username),
		Password:       info.Password,
		CACert:         info.CACert,
	}
}

// WebsocketE2ESuite is a suite that initialises a JIMM with
// an externally bootstrapped controller, and provides
// methods to open websocket connections to the JIMM API.
type WebsocketE2ESuite struct {
	E2ESuite

	Params     jujuapi.Params
	APIHandler http.Handler
	HTTP       *httptest.Server

	Credential2 *dbmodel.CloudCredential
	Model2      *dbmodel.Model
	Model3      *dbmodel.Model

	cancelFnc context.CancelFunc
}

type loginDetails struct {
	info          *api.Info
	username      string
	lp            api.LoginProvider
	dialWebsocket func(ctx context.Context, urlStr string, tlsConfig *tls.Config, ipAddr string) (jsoncodec.JSONConn, error)
}

func (s *WebsocketE2ESuite) SetUpTest(c *gc.C) {
	ctx, cancelFnc := context.WithCancel(context.Background())
	s.cancelFnc = cancelFnc

	s.E2ESuite.SetUpTest(c)

	s.Params.ControllerUUID = ControllerUUID

	mux := chi.NewRouter()
	mountHandler := func(path string, h jimmhttp.JIMMHttpHandler) {
		mux.Mount(path, h.Routes())
	}
	mux.Handle("/api", jujuapi.APIHandler(ctx, s.JIMM, s.Params))
	mountHandler(
		"/model/{uuid}/{type:charms|applications}",
		jimmhttp.NewHTTPProxyHandler(s.JIMM),
	)
	mux.Handle("/model/*", http.StripPrefix("/model", jujuapi.ModelHandler(ctx, s.JIMM, s.Params)))
	jwks := jimmhttp.NewWellKnownHandler(s.JIMM.CredentialStore)
	mux.HandleFunc("/.well-known/jwks.json", jwks.JWKS)
	mux.Handle("/migrate/logtransfer", jujuapi.LogTransferHandler(ctx, s.JIMM, s.Params))

	s.APIHandler = mux
	s.HTTP = httptest.NewTLSServer(s.APIHandler)

}

func (s *WebsocketE2ESuite) TearDownTest(c *gc.C) {
	if s.cancelFnc != nil {
		s.cancelFnc()
	}
	if s.HTTP != nil {
		s.HTTP.Close()
	}
	s.E2ESuite.TearDownTest(c)
}

// openNoAssert creates a new websocket connection to the test server, using the
// connection info specified in info, authenticating as the given user.
// If info is nil then default values will be used.
func (s *WebsocketE2ESuite) openNoAssert(c *gc.C, d loginDetails) (api.Connection, error) {
	var inf api.Info
	if d.info != nil {
		inf = *d.info
	}
	u, err := url.Parse(s.HTTP.URL)
	c.Assert(err, gc.Equals, nil)
	inf.Addrs = []string{
		u.Host,
	}
	w := new(bytes.Buffer)
	err = pem.Encode(w, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: s.HTTP.TLS.Certificates[0].Certificate[0],
	})
	c.Assert(err, gc.Equals, nil)
	inf.CACert = w.String()

	if d.lp == nil {
		d.lp = NewUserSessionLogin(c, d.username)
	}

	dialOpts := api.DialOpts{
		InsecureSkipVerify: true,
		LoginProvider:      d.lp,
	}

	if d.dialWebsocket != nil {
		dialOpts.DialWebsocket = d.dialWebsocket
	}

	return api.Open(&inf, dialOpts)
}

func (s *WebsocketE2ESuite) Open(c *gc.C, info *api.Info, username string) api.Connection {
	ld := loginDetails{info: info, username: username}
	conn, err := s.openNoAssert(c, ld)
	c.Assert(err, gc.Equals, nil)
	return conn
}

// E2ESuite is a suite that initialises a JIMM with
// an externally bootstrapped controller.
// It creates cloud credential, and creates a model.
type E2ESuite struct {
	JIMMSuite
	LoggingSuite

	CloudCredential *dbmodel.CloudCredential
	Model           *dbmodel.Model
}

func (s *E2ESuite) SetUpSuite(c *gc.C) {
	s.LoggingSuite.SetUpSuite(c)
}

func (s *E2ESuite) TearDownSuite(c *gc.C) {
	s.LoggingSuite.TearDownSuite(c)
}

func (s *E2ESuite) SetUpTest(c *gc.C) {
	s.UseHardcodedJWKS(c)
	s.JIMMSuite.SetUpTest(c)
	s.LoggingSuite.SetUpTest(c)

	// Add all controllers from the config file
	config := GetControllersConfig(c)
	for name, info := range config.Controllers {
		info.Validate(c, name)
		s.AddController(c, name, info.ToAPIInfo())
	}

	cct := names.NewCloudCredentialTag(TestE2ECloudName + "/bob@canonical.com/cred")
	cred := getExistingClientCredentialsForCloud(c, TestE2ECloudName).AuthCredentials
	cloudCredentials := jujuparams.CloudCredential{}
	for _, cred := range cred {
		cloudCredentials.AuthType = string(cred.AuthType())
		cloudCredentials.Attributes = cred.Attributes()
		break
	}
	s.UpdateCloudCredential(c, cct, cloudCredentials)
	ctx := context.Background()
	s.CloudCredential = new(dbmodel.CloudCredential)
	s.CloudCredential.SetTag(cct)
	err := s.JIMM.Database.GetCloudCredential(ctx, s.CloudCredential)
	c.Assert(err, gc.Equals, nil)

	mt := s.AddModel(c, names.NewUserTag("bob@canonical.com"), "model-1", names.NewCloudTag(TestE2ECloudName), TestE2ECloudName, cct)
	s.Model = new(dbmodel.Model)
	s.Model.SetTag(mt)
	err = s.JIMM.Database.GetModel(ctx, s.Model)
	c.Assert(err, gc.Equals, nil)
}

func (s *E2ESuite) TearDownTest(c *gc.C) {
	if s.Model != nil {
		s.DestroyModelAndDeleteFromDatabase(c, s.Model.ResourceTag())
	}
	s.JIMMSuite.TearDownTest(c)
}

func getExistingClientCredentialsForCloud(c *gc.C, cloudName string) cloud.CloudCredential {
	store := jclient.NewFileClientStore()
	existingCredentials, err := store.AllCredentials()
	c.Assert(err, gc.IsNil)
	cred, ok := existingCredentials[cloudName]
	c.Assert(ok, gc.Equals, true)
	return cred
}
