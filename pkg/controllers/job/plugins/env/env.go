/*
Copyright 2019 The Volcano Authors.

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

package env

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
	"strconv"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	jobhelpers "volcano.sh/volcano/pkg/controllers/job/helpers"
	pluginsinterface "volcano.sh/volcano/pkg/controllers/job/plugins/interface"
)

type envPlugin struct {
	// Arguments given for the plugin
	pluginArguments []string

	Clientset pluginsinterface.PluginClientset
}

// New creates env plugin.
func New(client pluginsinterface.PluginClientset, arguments []string) pluginsinterface.PluginInterface {
	envPlugin := envPlugin{pluginArguments: arguments, Clientset: client}

	return &envPlugin
}

func (ep *envPlugin) Name() string {
	return "env"
}

func (ep *envPlugin) OnPodCreate(pod *v1.Pod, job *batch.Job) error {
	index := jobhelpers.GetTaskIndex(pod)

	// Sum the number of pods in all different states + 1
	nextPodNum := 1 + job.Status.Pending + job.Status.Running +
		job.Status.Failed + job.Status.Terminating + job.Status.Succeeded + job.Status.Unknown

	counterLabel, counterLabelFound := pod.Annotations["volcano.sh/counter-label"]
	if counterLabelFound {
		klog.V(3).Infof("Job <%s> using counter label <%s>\n", job.Name, counterLabel)
	} else {
		klog.V(3).Infof("Job <%s> not using counter label\n", job.Name)
	}

	// add VK_TASK_INDEX and VC_TASK_INDEX env to each container
	for i := range pod.Spec.Containers {
		pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, v1.EnvVar{Name: TaskVkIndex, Value: index})
		pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, v1.EnvVar{Name: TaskIndex, Value: index})
		pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, v1.EnvVar{Name: TaskIndex, Value: index})

		// If a counter label name is specified on a pod
		// add a lebel to each container in the pod
		// where name = counter label name, value = nextPodNum
		if counterLabelFound {
			AddCounterLabel(pod, counterLabel, nextPodNum)
		}
	}

	// add VK_TASK_INDEX and VC_TASK_INDEX env to each init container
	for i := range pod.Spec.InitContainers {
		pod.Spec.InitContainers[i].Env = append(pod.Spec.InitContainers[i].Env, v1.EnvVar{Name: TaskVkIndex, Value: index})
		pod.Spec.InitContainers[i].Env = append(pod.Spec.InitContainers[i].Env, v1.EnvVar{Name: TaskIndex, Value: index})
	}

	return nil
}

func AddCounterLabel(pod *v1.Pod, counterLabel string, currentValue int32) {
	existingCounter, exists := pod.Labels[counterLabel]
	intExistingCounter, _ := strconv.Atoi(existingCounter)

	// Somehow the label always exists with value 0, so we only update the value if it is > 0
	if (!exists) || (intExistingCounter == 0) {
		klog.V(3).Infof("Setting counter label <%s> of Pod <%s> to <%d>\n", counterLabel,
			pod.Name, currentValue)
		pod.Labels[counterLabel] = strconv.Itoa(int(currentValue))
	} else {
		klog.V(3).Infof("pod <%s> has existing counter label <%s> with value <%s>\n",
			pod.Name, counterLabel, existingCounter)
	}

}
func (ep *envPlugin) OnJobAdd(job *batch.Job) error {
	if job.Status.ControlledResources["plugin-"+ep.Name()] == ep.Name() {
		return nil
	}

	job.Status.ControlledResources["plugin-"+ep.Name()] = ep.Name()

	return nil
}

func (ep *envPlugin) OnJobDelete(job *batch.Job) error {
	if job.Status.ControlledResources["plugin-"+ep.Name()] != ep.Name() {
		return nil
	}
	delete(job.Status.ControlledResources, "plugin-"+ep.Name())
	return nil
}

func (ep *envPlugin) OnJobUpdate(job *batch.Job) error {
	return nil
}
