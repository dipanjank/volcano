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

package app

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"sigs.k8s.io/yaml"
	"strconv"
	"syscall"

	"k8s.io/klog"

	v1 "k8s.io/api/core/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"volcano.sh/apis/pkg/apis/scheduling/scheme"
	"volcano.sh/volcano/cmd/webhook-manager/app/options"
	"volcano.sh/volcano/pkg/kube"
	"volcano.sh/volcano/pkg/version"
	"volcano.sh/volcano/pkg/webhooks/router"
)

// HierarchyWeights user configured weights for the queue hierarchy
type HierarchyWeights struct {
	Weights map[string]int
}

// readQueueConfig Read Dynamic Queue Configuration from a file.
func readQueueConfig(filePath string) map[string]int32 {
	hierarchyWeights := make(map[string]int32)
	klog.V(3).Infof("Trying to read Hierarchy weights from <%s>", filePath)
	contentBytes, err := ioutil.ReadFile(filePath)

	if err != nil {
		klog.Errorf("Queue config file <%s> does not exist or cannot be read: <%s>", filePath, err.Error())
		return hierarchyWeights
	}

	// Try to unmarshall the YAML
	weightsConf := &HierarchyWeights{}
	err = yaml.Unmarshal(contentBytes, weightsConf)

	if err != nil {
		klog.Errorf("Parse error in Queue config file <%s>: <%s>", filePath, err.Error())
		return hierarchyWeights
	}

	for nodeName, nodeWeight := range weightsConf.Weights {
		hierarchyWeights[nodeName] = int32(nodeWeight)
		klog.V(3).Infof("Using hierarchy weight <%d> for <%s>", nodeWeight, nodeName)
	}
	return hierarchyWeights
}

// Run start the service of admission controller.
func Run(config *options.Config) error {
	if config.PrintVersion {
		version.PrintVersionAndExit()
		return nil
	}

	if config.WebhookURL == "" && config.WebhookNamespace == "" && config.WebhookName == "" {
		return fmt.Errorf("failed to start webhooks as both 'url' and 'namespace/name' of webhook are empty")
	}

	restConfig, err := kube.BuildConfig(config.KubeClientOptions)
	if err != nil {
		return fmt.Errorf("unable to build k8s config: %v", err)
	}

	caBundle, err := ioutil.ReadFile(config.CaCertFile)
	if err != nil {
		return fmt.Errorf("unable to read cacert file (%s): %v", config.CaCertFile, err)
	}

	vClient := getVolcanoClient(restConfig)
	kubeClient := getKubeClient(restConfig)
	queueConfig := readQueueConfig(config.QueueConfigFile)

	// Fail the admission container if no hierarchy weights are found
	// This is not compatible with the dap use-case
	if len(queueConfig) == 0 {
		return fmt.Errorf("no queue hierarchy weights found in <%s>. Plase check admission configuration",
			config.QueueConfigFile)
	}

	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&corev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: config.SchedulerName})
	router.ForEachAdmission(func(service *router.AdmissionService) {
		if service.Config != nil {
			service.Config.VolcanoClient = vClient
			service.Config.SchedulerName = config.SchedulerName
			service.Config.Recorder = recorder
			service.Config.QueueConfig = queueConfig
		}

		klog.V(3).Infof("Registered '%s' as webhook.", service.Path)
		http.HandleFunc(service.Path, service.Handler)

		klog.V(3).Infof("Registered configuration for webhook <%s>", service.Path)
		registerWebhookConfig(kubeClient, config, service, caBundle)
	})

	webhookServeError := make(chan struct{})
	stopChannel := make(chan os.Signal)
	signal.Notify(stopChannel, syscall.SIGTERM, syscall.SIGINT)

	server := &http.Server{
		Addr:      ":" + strconv.Itoa(config.Port),
		TLSConfig: configTLS(config, restConfig),
	}
	go func() {
		err = server.ListenAndServeTLS("", "")
		if err != nil && err != http.ErrServerClosed {
			klog.Fatalf("ListenAndServeTLS for admission webhook failed: %v", err)
			close(webhookServeError)
		}

		klog.Info("Volcano Webhook manager started.")
	}()

	select {
	case <-stopChannel:
		if err := server.Close(); err != nil {
			return fmt.Errorf("close admission server failed: %v", err)
		}
		return nil
	case <-webhookServeError:
		return fmt.Errorf("unknown webhook server error")
	}
}
