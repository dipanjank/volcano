package mutate

import (
	"encoding/json"
	"fmt"
	"k8s.io/api/admission/v1beta1"
	whv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"volcano.sh/volcano/pkg/webhooks/router"
	"volcano.sh/volcano/pkg/webhooks/schema"
	"volcano.sh/volcano/pkg/webhooks/util"
)

func init() {
	router.RegisterAdmission(service)
}

var service = &router.AdmissionService{
	Path: "/pods/mutate",
	Func: Pods,

	Config: config,

	MutatingConfig: &whv1beta1.MutatingWebhookConfiguration{
		Webhooks: []whv1beta1.MutatingWebhook{{
			Name: "mutatepod.volcano.sh",
			Rules: []whv1beta1.RuleWithOperations{
				{
					Operations: []whv1beta1.OperationType{whv1beta1.Create},
					Rule: whv1beta1.Rule{
						APIGroups:   []string{""},
						APIVersions: []string{"v1alpha1"},
						Resources:   []string{"pods"},
					},
				},
			},
			ObjectSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{fmt.Sprintf("%s/%s", selectorPrefix, schedulerSelectorName): schedulerSelectorValue},
			},
		}},
	},
}

var config = &router.AdmissionServiceConfig{}

var selectorPrefix = config.AdditionalSelectors.SelectorPrefix
var schedulerSelectorName = config.AdditionalSelectors.SchedulerSelector.Name
var schedulerSelectorValue = config.AdditionalSelectors.SchedulerSelector.Value
var nodeSelectorName = config.AdditionalSelectors.NodeSelector.Name
var nodeSelectorValue = config.AdditionalSelectors.SchedulerSelector.Value

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// MutateJobs mutate jobs.
func Pods(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	klog.V(3).Infof("mutating pods")

	pod, err := schema.DecodePod(ar.Request.Object, ar.Request.Resource)
	if err != nil {
		return util.ToAdmissionResponse(err)
	}

	var patchBytes []byte
	switch ar.Request.Operation {
	case v1beta1.Create:
		patchBytes, _ = createPodPatch(pod)
	default:
		return util.ToAdmissionResponse(fmt.Errorf("invalid operation `%s`, "+
			"expect operation to be `CREATE`", ar.Request.Operation))
	}

	klog.V(3).Infof("AdmissionResponse: patch=%v", string(patchBytes))

	if err != nil {
		return &v1beta1.AdmissionResponse{
			Allowed: false,
			Result:  &metav1.Status{Message: err.Error()},
		}
	}

	pt := v1beta1.PatchTypeJSONPatch

	return &v1beta1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &pt,
	}
}

func createPodPatch(pod *v1.Pod) ([]byte, error) {
	var patch []patchOperation
	patchNodeSelector := patchNodeSelector(pod)
	if patchNodeSelector != nil {
		patch = append(patch, *patchNodeSelector)
	}
	return json.Marshal(patch)
}

func patchNodeSelector(pod *v1.Pod) *patchOperation {
	// Add dedicated node label.
	if nodeSelectorName != "" {
		op := "add"
		if pod.Spec.NodeSelector != nil {
			op = "replace"
		}
		path := fmt.Sprintf("/spec/nodeSelector/%s", nodeSelectorName)
		return &patchOperation{Op: op, Path: path, Value: nodeSelectorValue}
	}
	return nil
}