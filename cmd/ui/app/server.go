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
	"context"
	"embed"
	_ "embed"
	"fmt"
	"github.com/Masterminds/sprig"
	prometheusAPI "github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"html/template"
	_ "html/template"
	"io/fs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"net/http"
	"time"
	"volcano.sh/apis/pkg/apis/batch/v1alpha1"
	"volcano.sh/apis/pkg/apis/scheduling/v1beta1"
	"volcano.sh/apis/pkg/client/clientset/versioned"
	"volcano.sh/volcano/cmd/ui/app/options"
	"volcano.sh/volcano/pkg/kube"
	"volcano.sh/volcano/pkg/version"
)

var (
	vClient *versioned.Clientset
	kubeClient *kubernetes.Clientset
	prometheusApi prometheusv1.API
	prometheusCtx context.Context
	//go:embed templates/index.html
	index string
)


// Run start the service of admission controller.
func Run(config *options.Config) error {
	if config.PrintVersion {
		version.PrintVersionAndExit()
		return nil
	}

	if config.UiURL == "" && config.UiNamespace == "" && config.UiName == "" {
		return fmt.Errorf("failed to start uis as both 'url' and 'namespace/name' of ui are empty")
	}

	restConfig, err := kube.BuildConfig(config.KubeClientOptions)
	if err != nil {
		return fmt.Errorf("unable to build k8s config: %v", err)
	}

	client, err := prometheusAPI.NewClient(prometheusAPI.Config{
		Address: config.PrometheusURL,
	})
	if err != nil {
		return fmt.Errorf("unable to connect to prometheus: %v", err)
	}
	prometheusApi = prometheusv1.NewAPI(client)

	//caBundle, err := ioutil.ReadFile(config.CaCertFile)
	//if err != nil {
	//	return fmt.Errorf("unable to read cacert file (%s): %v", config.CaCertFile, err)
	//}

	vClient = getVolcanoClient(restConfig)
	kubeClient = getKubeClient(restConfig)

	http.HandleFunc("/", handler)
	http.Handle("/static/", http.StripPrefix("/static/",http.FileServer(getFileSystem())))

	return http.ListenAndServe(":8080", nil)
}

type Page struct {
	Queues *v1beta1.QueueList `json:"queues,omitempty"`
	Jobs   *v1alpha1.JobList  `json:"jobs,omitempty"`
	Metrics map[string]model.Value
}

//go:embed static
var embeddedFiles embed.FS

func getFileSystem() http.FileSystem {
	fsys, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		panic(err)
	}
	return http.FS(fsys)
}

type PromResult struct {
	QueueName string `yaml:"queue_name,omitempty"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	queues, _ := vClient.SchedulingV1beta1().Queues().List(context.TODO(), metav1.ListOptions{})
	jobs, _ := vClient.BatchV1alpha1().Jobs("default").List(context.TODO(), metav1.ListOptions{})
	query := []string{
		"volcano_queue_allocated_memory_bytes",
		"volcano_queue_allocated_milli_cpu",
	}
	metrics := make(map[string]model.Value)
	for _, param := range query {
		value, _, err := prometheusApi.Query(context.TODO(), param, time.Now())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		vector := value.(model.Vector)
		metrics[param] = vector
	}

	base, err := template.New("base").Funcs(sprig.FuncMap()).Parse(index)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t := template.Must(base, err)

	err = t.Execute(w, &Page{Queues: queues, Jobs: jobs, Metrics: metrics})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
