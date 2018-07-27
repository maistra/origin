package openshift

import (
	"bytes"
	"fmt"

	"github.com/blang/semver"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/glog"

	"github.com/openshift/origin/pkg/cmd/util/variable"
	"github.com/openshift/origin/pkg/oc/bootstrap/docker/errors"
	"github.com/openshift/origin/pkg/oc/cli/util/clientcmd"
)

const (
	istioPlaybook = "playbooks/openshift-istio/config.yml"
	istioNamespace = "istio-system"
)

// InstallIstio checks whether istio is installed and installs it if not already installed
func (h *Helper) InstallIstio(f *clientcmd.Factory, serverVersion semver.Version, serverIP,
		publicHostname, oseVersion, istioVersion, istioPrefix, istioJaegerVersion, istioElasticsearchMemory,
		hostConfigDir, imageStreams string, installCommunity, installAuth, installLauncher bool,
		launcherOpenShiftUser, launcherOpenShiftPassword, launcherGitHubUsername, launcherGitHubToken,
		launcherCatalogGitRepo, launcherCatalogGitBranch, launcherBoosterCatalogFilter string,
		istioKialiPrefix, istioKialiVersion, istioKialiUsername, istioKialiPassword string) error {
	kubeClient, err := f.ClientSet()
	if err != nil {
		return errors.NewError("cannot obtain API clients").WithCause(err).WithDetails(h.OriginLog())
	}
	securityClient, err := f.OpenshiftInternalSecurityClient()
	if err != nil {
		return errors.NewError("cannot obtain API clients").WithCause(err).WithDetails(h.OriginLog())
	}

	_, err = kubeClient.Core().Namespaces().Get(istioNamespace, metav1.GetOptions{})
	if err == nil {
		// If there's no error, the istio namespace already exists and we won't initialize it
		return nil
	}

	// Create istio namespace
	out := &bytes.Buffer{}
	err = CreateProject(f, istioNamespace, "", "", "oc", out)
	if err != nil {
		return errors.NewError("cannot create istio project").WithCause(err).WithDetails(out.String())
	}

	if (istioPrefix == variable.DefaultIstioImagePrefix) && (imageStreams != imageStreamCentos) {
		istioPrefix = "openshift3-istio-tech-preview/"
	}
	params := newAnsibleInventoryParams()
	params.Template = defaultIstioInventory
	params.MasterIP = serverIP
	params.MasterPublicURL = fmt.Sprintf("https://%s:8443", publicHostname)
	params.OSERelease = oseVersion
	params.IstioImageVersion = istioVersion
	params.IstioImagePrefix = istioPrefix
	params.IstioInstallCommunity = installCommunity
	params.IstioInstallAuth = installAuth
	params.IstioInstallLauncher = installLauncher
	params.IstioNamespace = istioNamespace
	params.IstioJaegerImageVersion = istioJaegerVersion
	params.IstioElasticsearchMemory = istioElasticsearchMemory
	params.LauncherOpenShiftUser = launcherOpenShiftUser
	params.LauncherOpenShiftPassword = launcherOpenShiftPassword
	params.LauncherGitHubUsername = launcherGitHubUsername
	params.LauncherGitHubToken = launcherGitHubToken
	params.LauncherCatalogGitRepo = launcherCatalogGitRepo
	params.LauncherCatalogGitBranch = launcherCatalogGitBranch
	params.LauncherBoosterCatalogFilter = launcherBoosterCatalogFilter
	params.IstioKialiImagePrefix = istioKialiPrefix
	params.IstioKialiImageVersion = istioKialiVersion
	params.IstioKialiUsername = istioKialiUsername
	params.IstioKialiPassword = istioKialiPassword

	if (istioKialiUsername == "") || (istioKialiPassword == "") {
		glog.Info("The kiali console will not be installed, please specify both istio-kiali-username and istio-kiali-password if desired")
	}
	runner := newAnsibleRunner(h, kubeClient, securityClient, istioNamespace, imageStreams, "istio")

	// Change this back once we have the productised installer
	playbookPrefix := "openshiftistio/origin"

	fmt.Fprintf(out, "Initialising Istio  using %s:%s ...\n", playbookPrefix, istioVersion)

	//run istio playbook
	return runner.RunPlaybook(params, istioPlaybook, hostConfigDir, playbookPrefix, istioVersion)
}
