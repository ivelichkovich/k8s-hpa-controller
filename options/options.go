//usr/bin/env go run $0 "$@"; exit
package options

import (
	"github.com/spf13/pflag"
)

type AutoScalerConfig struct {
	Debug             	bool
	PrometheusAddress 	string
	QueryExpression   	string
	PollPeriod        	string
	Namespace		  	string
	ScaleUpConstant   	float64
	ScaleUpThreshold  	float64
	ScaleDownConstant 	float64
	ScaleDownThreshold	float64
	ScaleDelay			int32
}

func NewAutoScalerConfig() *AutoScalerConfig {
	return &AutoScalerConfig{
		Debug:             false,
		PrometheusAddress: "https://prometheus.namespace",
		QueryExpression:   "sum(label_replace(rate(container_cpu_usage_seconds_total{pod_name=\"%s\"}[5m]),\"pod\",\"$1\",\"pod_name\",\"(.+)\"))/sum(kube_pod_container_resource_requests_cpu_cores{pod=\"%s\"})*100",
		PollPeriod:       	"60s",
		Namespace:			"default",
		ScaleUpConstant:	3,
		ScaleUpThreshold:	1.1,
		ScaleDownConstant:  1,
		ScaleDownThreshold: .9,
		ScaleDelay:			1,
	}
}

func (c *AutoScalerConfig) ValidateFlags() error {
	return nil
}

func (c *AutoScalerConfig) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&c.Debug, "debug", c.Debug, "Enable debug.")
	fs.StringVar(&c.PrometheusAddress, "prom-location", c.PrometheusAddress, "Prometheus location name and port number. http://prometheus")
	fs.StringVar(&c.QueryExpression, "query-exp", c.QueryExpression, "Expression used to query Prometheus for scaling, use %s to replace pod name")
	fs.StringVar(&c.PollPeriod, "poll-period", c.PollPeriod, "How often to poll prometheus, be careful because unless prometheus scrapes polls can be redundant, default is '60s'")
	fs.StringVar(&c.Namespace, "namespace", c.Namespace, "Namespace to Deploy to")
	fs.Float64Var(&c.ScaleUpConstant, "scale-up-constant", c.ScaleUpConstant, "Default 3, will scale up by ceiling(currentCPU/targetCPU)*currentReplicas tweaked by this constant")
	fs.Float64Var(&c.ScaleDownConstant, "scale-down-constant", c.ScaleDownConstant, "Default .5, will scale down by floor(currentCPU/targetCPU)*currentReplicas tweaked by this constant")
	fs.Float64Var(&c.ScaleUpThreshold, "scale-up-threshold", c.ScaleUpThreshold, "Default 1.1, will try to scale if currentCPU/targetCPU > this value")
	fs.Float64Var(&c.ScaleDownThreshold, "scale-down-threshold", c.ScaleDownThreshold, "Default 1, will try to scale if currentCPU/targetCPU < this value")
	fs.Int32Var(&c.ScaleDelay, "scale-delay", c.ScaleDelay, "Number of poll cycles to wait between scales, default of 1")
}
