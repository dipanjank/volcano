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

package options

import (
	"fmt"

	"github.com/spf13/pflag"

	"volcano.sh/volcano/pkg/kube"
)

const (
	defaultSchedulerName = "volcano"
	defaultQPS           = 50.0
	defaultBurst         = 100
	defaultPrometheusURL = "http://localhost:30003"
)

// Config admission-controller server config.
type Config struct {
	KubeClientOptions  kube.ClientOptions
	CertFile           string
	KeyFile            string
	CaCertFile         string
	Port               int
	PrintVersion       bool
	DashboardName      string
	DashboardNamespace string
	SchedulerName      string
	DashboardURL       string
	PrometheusURL      string
}

// NewConfig create new config.
func NewConfig() *Config {
	c := Config{}
	return &c
}

// AddFlags add flags.
func (c *Config) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.KubeClientOptions.Master, "master", c.KubeClientOptions.Master, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	fs.StringVar(&c.KubeClientOptions.KubeConfig, "kubeconfig", c.KubeClientOptions.KubeConfig, "Path to kubeconfig file with authorization and master location information.")
	fs.StringVar(&c.CertFile, "tls-cert-file", c.CertFile, ""+
		"File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated "+
		"after server cert).")
	fs.StringVar(&c.KeyFile, "tls-private-key-file", c.KeyFile, "File containing the default x509 private key matching --tls-cert-file.")
	fs.IntVar(&c.Port, "port", 8443, "the port used by admission-controller-server.")
	fs.BoolVar(&c.PrintVersion, "version", false, "Show version and quit")
	fs.Float32Var(&c.KubeClientOptions.QPS, "kube-api-qps", defaultQPS, "QPS to use while talking with kubernetes apiserver")
	fs.IntVar(&c.KubeClientOptions.Burst, "kube-api-burst", defaultBurst, "Burst to use while talking with kubernetes apiserver")

	fs.StringVar(&c.CaCertFile, "ca-cert-file", c.CaCertFile, "File containing the x509 Certificate for HTTPS.")
	fs.StringVar(&c.DashboardNamespace, "dashboard-namespace", "", "The namespace of this dashboard")
	fs.StringVar(&c.DashboardName, "dashboard-service-name", "", "The name of this dashboard")
	fs.StringVar(&c.DashboardURL, "dashboard-url", "", "The url of this dashboard")

	fs.StringVar(&c.SchedulerName, "scheduler-name", defaultSchedulerName, "Volcano will handle pods whose .spec.SchedulerName is same as scheduler-name")

	fs.StringVar(&c.PrometheusURL, "prometheus-url", defaultPrometheusURL, "Volcano will retrieve metrics from the prometheus")
}

// CheckPortOrDie check valid port range.
func (c *Config) CheckPortOrDie() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("the port should be in the range of 1 and 65535")
	}
	return nil
}
