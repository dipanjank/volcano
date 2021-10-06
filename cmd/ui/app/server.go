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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Masterminds/sprig"
	"github.com/jahnestacado/tlru"
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
	cache tlru.TLRU
	pageCache tlru.Entry
	ttl = 2 * time.Second
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

	evictionChannel := make(chan tlru.EvictedEntry, 0)
	tlruConfig := tlru.Config{
		Size:            2,
		TTL:             ttl,
		EvictionPolicy:  tlru.LRA,
		EvictionChannel: &evictionChannel,
	}

	cache = tlru.New(tlruConfig)

	go func() {
		fmt.Printf("go routine")
		for {
			evictedEntry := <-evictionChannel
			if evictedEntry.Reason.String() != "Expired" {
				fmt.Printf("Entry with key: '%s' has been evicted with reason: %s\n", evictedEntry.Key, evictedEntry.Reason.String())
			}
		}
	}()

	// set initial metrics
	page, err := getMetrics()

	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	pageCache = tlru.Entry{Key: "data", Value: page}

	err = cache.Set(pageCache)

	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	http.HandleFunc("/", pageHandler)
	http.HandleFunc("/metrics.json", dataHandler)
	http.Handle("/static/", http.StripPrefix("/static/",http.FileServer(getFileSystem())))

	return http.ListenAndServe(":8080", nil)
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

type Page struct {
	Queues *v1beta1.QueueList `json:"queues,omitempty"`
	Jobs   *v1alpha1.JobList  `json:"jobs,omitempty"`
	Metrics map[string]model.Value `json:"metrics,omitempty"`
}

func getMetrics() (*Page, error) {
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
			return nil, err
		}
		vector := value.(model.Vector)
		metrics[param] = vector
	}
	return &Page{Queues: queues, Jobs: jobs, Metrics: metrics}, nil
}

func getCachedMetrics() (*Page, error) {
	pageEntry := cache.Get(pageCache.Key)

	var page *Page

	if pageEntry == nil {
		fmt.Printf("retrieve metrics")
		page, err := getMetrics()

		if err != nil {
			return nil, err
		}
		pageCache = tlru.Entry{Key: "data", Value: page}

		err = cache.Set(pageCache)
		if err != nil {
			return nil, err
		}
	} else {
		var okay bool
		page, okay = pageEntry.Value.(*Page)
		if !okay {
			return nil, errors.New("type conversion error")
		}
	}

	return page, nil
}
func pageHandler(w http.ResponseWriter, r *http.Request) {

	page, err := getCachedMetrics()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	base, err := template.New("base").Funcs(sprig.FuncMap()).Parse(index)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t := template.Must(base, err)

	err = t.Execute(w, page)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func dataHandler(w http.ResponseWriter, r *http.Request) {
	page, err := getCachedMetrics()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.Encode(page)
}