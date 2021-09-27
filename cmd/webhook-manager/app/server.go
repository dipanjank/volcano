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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"reflect"
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

// readQueueConfig Read Dynamic Queue Configuration from a file.
func readQueueConfig(filePath string) map[string]int32 {
	hierarchyWeights := make(map[string]int32)

	jsonFile, err := os.Open(filePath)
	// if we os.Open returns an error then handle it
	if err != nil {
		klog.Warningf("Queue config file <%s> does not exist or cannot be read: <%s>", filePath, err.Error())
		return hierarchyWeights
	}

	// defer the closing of our jsonFile so that we can parse it later on
	defer func(jsonFile *os.File) {
		err := jsonFile.Close()
		if err != nil {
		}
	}(jsonFile)

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var result map[string]interface{}

	err = json.Unmarshal(byteValue, &result)
	if err != nil {
		klog.Warningf("Parse error in Queue config file <%s>: <%s>", filePath, err.Error())
		return hierarchyWeights
	}

	// Read the hierarchyweights element from the json and parse the values into int
	weights, exists := result["hierarchyweights"]

	if !exists {
		klog.Warningf("No entry <hierarchyweights> found in Queue config file <%s>", filePath)
		return hierarchyWeights
	}

	wIter := reflect.ValueOf(weights).MapRange()

	for wIter.Next() {
		name := fmt.Sprintf("%v", wIter.Key())
		weight, err := strconv.Atoi(fmt.Sprintf("%v", wIter.Value()))
		if err == nil {
			hierarchyWeights[name] = int32(weight)
		}
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
