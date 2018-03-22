package openshift

import (
	"bytes"
	"fmt"

	"github.com/blang/semver"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	kapi "k8s.io/kubernetes/pkg/apis/core"

	"github.com/openshift/origin/pkg/oc/bootstrap/docker/errors"
	"github.com/openshift/origin/pkg/oc/cli/util/clientcmd"
)

const (
	istioPlaybook = "playbooks/openshift-istio/config.yml"
	istioNamespace = "istio-system"
)

// InstallIstio checks whether istio is installed and installs it if not already installed
func (h *Helper) InstallIstio(f *clientcmd.Factory, serverVersion semver.Version, serverIP, publicHostname, oseVersion, istioVersion, istioPrefix, hostConfigDir, imageStreams string, installCommunity bool) error {
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

	params := newAnsibleInventoryParams()
	params.Template = defaultIstioInventory
	params.MasterIP = serverIP
	params.MasterPublicURL = fmt.Sprintf("https://%s:8443", publicHostname)
	params.OSERelease = oseVersion
	params.IstioImageVersion = istioVersion
	params.IstioImagePrefix = istioPrefix
	params.IstioInstallCommunity = installCommunity
	params.IstioNamespace = istioNamespace

	runner := newAnsibleRunner(h, kubeClient, securityClient, istioNamespace, imageStreams, "istio")

	playbookPrefix := "openshift"
	if imageStreams == imageStreamCentos {
		playbookPrefix = "origin"
	}

	fmt.Fprintf(out, "Initialising Istio  using %s:%s ...\n", fmt.Sprintf("%s%s", istioPrefix, playbookPrefix), istioVersion)

	//run istio playbook
	return runner.RunPlaybook(params, istioPlaybook, hostConfigDir, fmt.Sprintf("%s%s", istioPrefix, playbookPrefix), istioVersion)
}
