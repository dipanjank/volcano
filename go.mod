module volcano.sh/volcano

go 1.16

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/fsnotify/fsnotify v1.4.9
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/hashicorp/go-multierror v1.0.0
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/jahnestacado/tlru v0.0.0-20200730090107-e1ec5c96073e
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.7.0
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/common v0.10.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	go.uber.org/atomic v1.4.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.19.6
	k8s.io/apimachinery v0.19.6
	k8s.io/client-go v0.19.6
	k8s.io/component-base v0.19.6
	k8s.io/klog v1.0.0
	k8s.io/kube-scheduler v0.0.0
	k8s.io/kubernetes v1.19.6
	sigs.k8s.io/yaml v1.2.0
	volcano.sh/apis v0.0.0-20210518032656-21e2239e42bc
)

replace (
	k8s.io/api => k8s.io/api v0.19.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.6
	k8s.io/apiserver => k8s.io/apiserver v0.19.6
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.19.6
	k8s.io/client-go => k8s.io/client-go v0.19.6
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.19.6
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.19.6
	k8s.io/code-generator => k8s.io/code-generator v0.19.6
	k8s.io/component-base => k8s.io/component-base v0.19.6
	k8s.io/cri-api => k8s.io/cri-api v0.19.6
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.19.6
	k8s.io/klog => k8s.io/klog v1.0.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.19.6
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.19.6
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.19.6
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.19.6
	k8s.io/kubectl => k8s.io/kubectl v0.19.6
	k8s.io/kubelet => k8s.io/kubelet v0.19.6
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.19.6
	k8s.io/metrics => k8s.io/metrics v0.19.6
	k8s.io/node-api => k8s.io/node-api v0.19.6
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.19.6
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.19.6
	k8s.io/sample-controller => k8s.io/sample-controller v0.19.6
	volcano.sh/apis => github.com/Tincoboekestijn/apis v0.0.0-20210907095657-54a1ba9c4b64
)
