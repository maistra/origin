package istio

import (
	"fmt"
	"io/ioutil"
	"path"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/runtime"

	configapi "github.com/openshift/origin/pkg/cmd/server/apis/config"
	configapilatest "github.com/openshift/origin/pkg/cmd/server/apis/config/latest"
	"github.com/openshift/origin/pkg/cmd/util/variable"
	"github.com/openshift/origin/pkg/oc/clusteradd/componentinstall"
	"github.com/openshift/origin/pkg/oc/clusterup/coreinstall/kubeapiserver"
	"github.com/openshift/origin/pkg/oc/clusterup/docker/dockerhelper"
	"github.com/openshift/origin/pkg/oc/clusterup/manifests"
	"github.com/openshift/origin/pkg/version"
	"strings"
)

const (
	namespace = "istio-operator"
	defaultImageFormat = "maistra/${component}-centos7:${version}"
)

var (
	IstioVersion string
)

type IstioComponentOptions struct {
	InstallContext componentinstall.Context
}

func (c *IstioComponentOptions) Name() string {
	return "istio"
}

func (c *IstioComponentOptions) Install(dockerClient dockerhelper.Interface) error {
	imageTemplate := variable.NewDefaultImageTemplate()
	imageTemplate.Format = variable.Expand(defaultImageFormat, func(s string) (string, bool) {
		if s == "version" {
			return IstioVersion, true
		}
		return "", false
	}, variable.Identity)
	imageTemplate.Latest = false

	release := getRelease()
	masterPublicURL, err := getMasterPublicURL(c.InstallContext.BaseDir())
	if err != nil {
		return err
	}
	params := map[string]string{
		"IMAGE":             imageTemplate.ExpandOrDie("istio-operator"),
		"NAMESPACE":         namespace,
		"PULL_POLICY":       c.InstallContext.ImagePullPolicy(),
		"RELEASE":           release,
		"MASTER_PUBLIC_URL": masterPublicURL,
	}

	glog.V(2).Infof("instantiating istio operator template with parameters %v", params)

	component := componentinstall.Template{
		Name:            "istio-operator",
		Namespace:       namespace,
		RBACTemplate:    manifests.MustAsset("install/istio/install-rbac.yaml"),
		InstallTemplate: manifests.MustAsset("install/istio/install.yaml"),
	}
	err = component.MakeReady(
		c.InstallContext.ClientImage(),
		c.InstallContext.BaseDir(),
		params).Install(dockerClient)
	if err != nil {
		glog.Errorf("Failed to install Istio component: %v", err)
		return err
	}

	return nil
}

func getRelease() string {
	return strings.TrimRight("v"+version.Get().Major+"."+version.Get().Minor, "+")
}

func getMasterPublicURL(basedir string) (string, error) {
	masterConfig, err := getMasterConfig(basedir)
	if err != nil {
		return "", err
	}
	return masterConfig.MasterPublicURL, nil
}

func getMasterConfig(basedir string) (*configapi.MasterConfig, error) {
	configBytes, err := ioutil.ReadFile(path.Join(basedir, kubeapiserver.KubeAPIServerDirName, "master-config.yaml"))
	if err != nil {
		return nil, err
	}
	configObj, err := runtime.Decode(configapilatest.Codec, configBytes)
	if err != nil {
		return nil, err
	}
	masterConfig, ok := configObj.(*configapi.MasterConfig)
	if !ok {
		return nil, fmt.Errorf("the %#v is not MasterConfig", configObj)
	}
	return masterConfig, nil
}
