/*
Copyright 2018 The Volcano Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validate

import (
	"context"
	"fmt"
	"k8s.io/api/admission/v1beta1"
	whv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	v1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	k8score "k8s.io/kubernetes/pkg/apis/core"
	k8scorev1 "k8s.io/kubernetes/pkg/apis/core/v1"
	k8scorevalid "k8s.io/kubernetes/pkg/apis/core/validation"
	"strings"
	"time"
	"volcano.sh/apis/pkg/apis/batch/v1alpha1"
	schedulingv1beta1 "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
	jobhelpers "volcano.sh/volcano/pkg/controllers/job/helpers"
	"volcano.sh/volcano/pkg/controllers/job/plugins"
	"volcano.sh/volcano/pkg/webhooks/admission/jobs/mutate"
	"volcano.sh/volcano/pkg/webhooks/router"
	"volcano.sh/volcano/pkg/webhooks/schema"
	"volcano.sh/volcano/pkg/webhooks/util"
)

func init() {
	router.RegisterAdmission(service)
}

var service = &router.AdmissionService{
	Path: "/jobs/validate",
	Func: AdmitJobs,

	Config: config,

	ValidatingConfig: &whv1beta1.ValidatingWebhookConfiguration{
		Webhooks: []whv1beta1.ValidatingWebhook{{
			Name: "validatejob.volcano.sh",
			Rules: []whv1beta1.RuleWithOperations{
				{
					Operations: []whv1beta1.OperationType{whv1beta1.Create, whv1beta1.Update},
					Rule: whv1beta1.Rule{
						APIGroups:   []string{"batch.volcano.sh"},
						APIVersions: []string{"v1alpha1"},
						Resources:   []string{"jobs"},
					},
				},
			},
		}},
	},
}

var config = &router.AdmissionServiceConfig{}

// AdmitJobs is to validate jobs and return response.
func AdmitJobs(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	klog.V(3).Infof("admitting jobs -- %s", ar.Request.Operation)

	job, err := schema.DecodeJob(ar.Request.Object, ar.Request.Resource)
	if err != nil {
		return util.ToAdmissionResponse(err)
	}
	var msg string
	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true

	switch ar.Request.Operation {
	case v1beta1.Create:
		msg = validateJobCreate(job, &reviewResponse)
	case v1beta1.Update:
		oldJob, err := schema.DecodeJob(ar.Request.OldObject, ar.Request.Resource)
		if err != nil {
			return util.ToAdmissionResponse(err)
		}
		err = validateJobUpdate(oldJob, job)
		if err != nil {
			return util.ToAdmissionResponse(err)
		}
	default:
		err := fmt.Errorf("expect operation to be 'CREATE' or 'UPDATE'")
		return util.ToAdmissionResponse(err)
	}

	if !reviewResponse.Allowed {
		reviewResponse.Result = &metav1.Status{Message: strings.TrimSpace(msg)}
	}
	return &reviewResponse
}

func validateJobCreate(job *v1alpha1.Job, reviewResponse *v1beta1.AdmissionResponse) string {
	var msg string
	taskNames := map[string]string{}
	var totalReplicas int32

	if job.Spec.MinAvailable < 0 {
		reviewResponse.Allowed = false
		return fmt.Sprintf("job 'minAvailable' must be >= 0.")
	}

	if job.Spec.MaxRetry < 0 {
		reviewResponse.Allowed = false
		return fmt.Sprintf("'maxRetry' cannot be less than zero.")
	}

	if job.Spec.TTLSecondsAfterFinished != nil && *job.Spec.TTLSecondsAfterFinished < 0 {
		reviewResponse.Allowed = false
		return fmt.Sprintf("'ttlSecondsAfterFinished' cannot be less than zero.")
	}

	if len(job.Spec.Tasks) == 0 {
		reviewResponse.Allowed = false
		return fmt.Sprintf("No task specified in job spec")
	}

	for index, task := range job.Spec.Tasks {
		if task.Replicas < 0 {
			msg += fmt.Sprintf(" 'replicas' < 0 in task: %s;", task.Name)
		}

		if task.MinAvailable != nil && *task.MinAvailable > task.Replicas {
			msg += fmt.Sprintf(" 'minAvailable' is greater than 'replicas' in task: %s, job: %s", task.Name, job.Name)
		}

		// count replicas
		totalReplicas += task.Replicas

		// validate task name
		if errMsgs := validation.IsDNS1123Label(task.Name); len(errMsgs) > 0 {
			msg += fmt.Sprintf(" %v;", errMsgs)
		}

		// duplicate task name
		if _, found := taskNames[task.Name]; found {
			msg += fmt.Sprintf(" duplicated task name %s;", task.Name)
			break
		} else {
			taskNames[task.Name] = task.Name
		}

		if err := validatePolicies(task.Policies, field.NewPath("spec.tasks.policies")); err != nil {
			msg += err.Error() + fmt.Sprintf(" valid events are %v, valid actions are %v",
				getValidEvents(), getValidActions())
		}
		podName := jobhelpers.MakePodName(job.Name, task.Name, index)
		msg += validateK8sPodNameLength(podName)
		msg += validateTaskTemplate(task, job, index)
	}

	msg += validateJobName(job)

	if totalReplicas < job.Spec.MinAvailable {
		msg += "job 'minAvailable' should not be greater than total replicas in tasks;"
	}

	if err := validatePolicies(job.Spec.Policies, field.NewPath("spec.policies")); err != nil {
		msg = msg + err.Error() + fmt.Sprintf(" valid events are %v, valid actions are %v;",
			getValidEvents(), getValidActions())
	}

	// invalid job plugins
	if len(job.Spec.Plugins) != 0 {
		for name := range job.Spec.Plugins {
			if _, found := plugins.GetPluginBuilder(name); !found {
				msg += fmt.Sprintf(" unable to find job plugin: %s", name)
			}
		}
	}

	if err := validateIO(job.Spec.Volumes); err != nil {
		msg += err.Error()
	}

	if dynamicQueue, ok := job.Annotations["volcano.sh/dynamic-queue"]; ok {
		// dynamic queue name must start with "root", otherwise DRF does not work properly.
		queueHierarchy := strings.Split(dynamicQueue, "/")
		if queueHierarchy[0] == "root" {
			if err := createDynamicQueue(queueHierarchy); err != nil {
				msg += err.Error()
			}
		} else {
			msg += fmt.Sprintf("Dynamic Queue name <%s> does not start with root", dynamicQueue)
		}
	}

	queue, err := config.VolcanoClient.SchedulingV1beta1().Queues().Get(context.TODO(), job.Spec.Queue, metav1.GetOptions{})
	if err != nil {
		msg += fmt.Sprintf(" unable to find job queue: %v", err)
	} else if queue.Status.State != schedulingv1beta1.QueueStateOpen {
		msg += fmt.Sprintf("can only submit job to queue with state `Open`, "+
			"queue `%s` status is `%s`", queue.Name, queue.Status.State)
	}

	if msg != "" {
		reviewResponse.Allowed = false
	}

	return msg
}

// createDynamicQueue Dynamically create the queue hierarchy node by node, checking if each node exists and creating
// if it does not. Construct the hierarchy weight by concatenating the hierarchy weights defined in configuration.
func createDynamicQueue(queueHierarchy []string) error {

	for _, nodeName := range queueHierarchy {
		if nodeName == mutate.DefaultQueue {
			return errors.NewBadRequest("Cannot use default queue as part of the hierarchy.")
		}
	}

	n := len(queueHierarchy)
	queueName := queueHierarchy[n-1]

	// Check if the queue already exists
	_, getError := config.VolcanoClient.SchedulingV1beta1().Queues().Get(
		context.TODO(),
		queueName,
		metav1.GetOptions{})

	if errors.IsNotFound(getError) {
		hierarchy := strings.Join(queueHierarchy, "/")

		// Construct the hierarchy weight by checking if each node in the path has a weight specified
		// in the config. If not found, use 1 as default
		var hierarchyWeights []string

		for _, name := range queueHierarchy {
			weight, exists := config.QueueConfig[name]
			if exists {
				hierarchyWeights = append(hierarchyWeights, fmt.Sprintf("%d", weight))
			} else {
				hierarchyWeights = append(hierarchyWeights, "1")
			}
		}
		_, err := createQueue(queueName, hierarchy, strings.Join(hierarchyWeights, "/"))
		if err != nil {
			return err
		}
	} else {
		klog.V(3).Infof("Queue <%s> already exists.", queueName)
	}
	return nil
}

func createQueue(queueName string, hierarchy string, hierarchyWeights string) (*schedulingv1beta1.Queue, error) {
	var queue = schedulingv1beta1.Queue{
		ObjectMeta: metav1.ObjectMeta{
			Name: queueName,
			Annotations: map[string]string{
				"volcano.sh/hierarchy":         hierarchy,
				"volcano.sh/hierarchy-weights": hierarchyWeights,
			},
		},
		Spec: schedulingv1beta1.QueueSpec{
			Weight: 1,
		},
		Status: schedulingv1beta1.QueueStatus{
			State: schedulingv1beta1.QueueStateOpen,
		},
	}

	klog.V(3).Infof("Trying to create queue <%s> with hierarchy <%s>, weights <%s>",
		queueName, hierarchy, hierarchyWeights)
	if _, err := config.VolcanoClient.SchedulingV1beta1().Queues().Create(context.TODO(), &queue, metav1.CreateOptions{}); err != nil {
		klog.Errorf("unable to create the dynamic queue <`%s`>: %v", queueName, err)
		return nil, err
	}

	if err := wait.PollImmediate(time.Second/2, time.Second*5, isQueueRunning(queueName)); err != nil {
		klog.Errorf("unable to use the dynamic queue <`%s`>: %v", queueName, err)
		return nil, err
	}

	return &queue, nil
}

func isQueueRunning(queueName string) wait.ConditionFunc {
	return func() (bool, error) {
		queue, err := config.VolcanoClient.SchedulingV1beta1().Queues().Get(context.TODO(), queueName, metav1.GetOptions{})

		if err != nil {
			return false, err
		}

		if queue.Status.State == schedulingv1beta1.QueueStateOpen {
			return true, nil
		}

		return false, nil
	}
}

func validateJobUpdate(old, new *v1alpha1.Job) error {
	var totalReplicas int32
	for _, task := range new.Spec.Tasks {
		if task.Replicas < 0 {
			return fmt.Errorf("'replicas' must be >= 0 in task: %s", task.Name)
		}

		if task.MinAvailable != nil && *task.MinAvailable > task.Replicas {
			return fmt.Errorf("'minAvailable' must be <= 'replicas' in task: %s;", task.Name)
		}
		// count replicas
		totalReplicas += task.Replicas
	}
	if new.Spec.MinAvailable > totalReplicas {
		return fmt.Errorf("job 'minAvailable' must not be greater than total replicas")
	}
	if new.Spec.MinAvailable < 0 {
		return fmt.Errorf("job 'minAvailable' must be >= 0")
	}

	if len(old.Spec.Tasks) != len(new.Spec.Tasks) {
		return fmt.Errorf("job updates may not add or remove tasks")
	}
	// other fields under spec are not allowed to mutate
	new.Spec.MinAvailable = old.Spec.MinAvailable
	new.Spec.PriorityClassName = old.Spec.PriorityClassName
	for i := range new.Spec.Tasks {
		new.Spec.Tasks[i].Replicas = old.Spec.Tasks[i].Replicas
		new.Spec.Tasks[i].MinAvailable = old.Spec.Tasks[i].MinAvailable
	}

	// job controller will update the pvc name if not provided
	for i := range new.Spec.Volumes {
		if new.Spec.Volumes[i].VolumeClaim != nil {
			new.Spec.Volumes[i].VolumeClaimName = ""
		}
	}
	for i := range old.Spec.Volumes {
		if old.Spec.Volumes[i].VolumeClaim != nil {
			old.Spec.Volumes[i].VolumeClaimName = ""
		}
	}

	if !apiequality.Semantic.DeepEqual(new.Spec, old.Spec) {
		return fmt.Errorf("job updates may not change fields other than `minAvailable`, `tasks[*].replicas under spec`")
	}

	return nil
}

func validateTaskTemplate(task v1alpha1.TaskSpec, job *v1alpha1.Job, index int) string {
	var v1PodTemplate v1.PodTemplate
	v1PodTemplate.Template = *task.Template.DeepCopy()
	k8scorev1.SetObjectDefaults_PodTemplate(&v1PodTemplate)

	var coreTemplateSpec k8score.PodTemplateSpec
	k8scorev1.Convert_v1_PodTemplateSpec_To_core_PodTemplateSpec(&v1PodTemplate.Template, &coreTemplateSpec, nil)

	// Skip verify container SecurityContex.Privileged as it depends on
	// the kube-apiserver `allow-privileged` flag.
	for i, container := range coreTemplateSpec.Spec.Containers {
		if container.SecurityContext != nil && container.SecurityContext.Privileged != nil {
			coreTemplateSpec.Spec.Containers[i].SecurityContext.Privileged = nil
		}
	}

	corePodTemplate := k8score.PodTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      task.Name,
			Namespace: job.Namespace,
		},
		Template: coreTemplateSpec,
	}

	if allErrs := k8scorevalid.ValidatePodTemplate(&corePodTemplate); len(allErrs) > 0 {
		msg := fmt.Sprintf("spec.task[%d].", index)
		for index := range allErrs {
			msg += allErrs[index].Error() + ". "
		}
		return msg
	}

	return ""
}

func validateK8sPodNameLength(podName string) string {
	if errMsgs := validation.IsQualifiedName(podName); len(errMsgs) > 0 {
		return fmt.Sprintf("create pod with name %s validate failed %v;", podName, errMsgs)
	}
	return ""
}

func validateJobName(job *v1alpha1.Job) string {
	if errMsgs := validation.IsQualifiedName(job.Name); len(errMsgs) > 0 {
		return fmt.Sprintf("create job with name %s validate failed %v", job.Name, errMsgs)
	}
	return ""
}
