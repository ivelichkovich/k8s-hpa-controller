//usr/bin/env go run $0 "$@"; exit
package autoscaler

import (
	"fmt"
	"io/ioutil"
        "net/http"
	//"os"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/util/clock"
	"github.com/tidwall/gjson"
	"github.com/golang/glog"
	"github.com/ivelichkovich/k8s-hpa-controller/options"
	"k8s.io/client-go/pkg/api/v1"
	meta_v1 "k8s.io/client-go/pkg/apis/meta/v1"
	"strconv"
	"math"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"strings"
)

type AutoScaler struct {
	debug				bool
	clientset      		*kubernetes.Clientset
	prometheus			string
	queryExp			string
	pollPeriod    		time.Duration
	clock         		clock.Clock
	stopCh        		chan struct{}
	HpaEntities			[]HpaEntity
	recentlyScaled 		int32
	Namespace 			string
	ScaleUpConstant   	float64
	ScaleUpThreshold  	float64
	ScaleDownConstant 	float64
	ScaleDownThreshold	float64
	ScaleDelay			int32
}

type HpaEntity struct {
	MaxReplicas			int32
	MinReplicas			int32
	TargetCPU			int32
	CurrentCPU			float64
	Deployment			*v1beta1.Deployment
	CurrentReplicas		int32
}

func NewAutoScaler(c *options.AutoScalerConfig, clientset *kubernetes.Clientset) (*AutoScaler, error) {
	thisPollPeriod, err := time.ParseDuration(c.PollPeriod)
	if err != nil {
		return nil, err
	}
	if time.Second > thisPollPeriod {
		thisPollPeriod = time.Second
	}
	return &AutoScaler{
		debug:				c.Debug,
		clientset:     		clientset,
		prometheus:	   		c.PrometheusAddress,
		queryExp:	   		c.QueryExpression,
		pollPeriod:    		thisPollPeriod,
		clock:         		clock.RealClock{},
		stopCh:        		make(chan struct{}),
		Namespace:     		c.Namespace,
		ScaleUpConstant:	c.ScaleUpConstant,
		ScaleUpThreshold:	c.ScaleUpThreshold,
		ScaleDownConstant:	c.ScaleDownConstant,
		ScaleDownThreshold:	c.ScaleDownThreshold,
		ScaleDelay:			c.ScaleDelay,
	}, nil
}

func (s *AutoScaler) Run() {
	ticker := s.clock.Tick(s.pollPeriod)

	s.pollCpuUsage()

	for {
		select {
		case <-ticker:
			if s.recentlyScaled < 1 {
				fmt.Println("Polling")
				s.pollCpuUsage()
			} else {
				s.recentlyScaled--
			}
		case <-s.stopCh:
			return
		}
	}
}

func (s *AutoScaler) generateHpaEntities() {
	debug := s.debug
	namespace := s.Namespace
	hpas, err := s.clientset.HorizontalPodAutoscalers(namespace).List(v1.ListOptions{})
	if err != nil {
		glog.Fatal(err)
	}

	var hpaEntities []HpaEntity
	// loop through hpas in namespace
	for _, hpa := range hpas.Items {
		// grab the deployment
		deploymentName := hpa.Spec.ScaleTargetRef.Name
		if debug {
			fmt.Printf("Found deployment for hpa %s: as %s", hpa.Name, deploymentName)
		}
		deployment, err := s.clientset.Deployments(namespace).Get(deploymentName, meta_v1.GetOptions{})
		if err != nil {
			glog.Fatal(err)
		}
		// grab the pods for the deployment
		podListOptions := v1.ListOptions{
			LabelSelector: fmt.Sprintf("app=%s", deployment.ObjectMeta.Labels["app"]),
		}
		pods, err := s.clientset.Pods(namespace).List(podListOptions)
		if err != nil {
			glog.Fatal(err)
		}

		podCpuUsages := make([]float64, len(pods.Items))
		// find cpu usage (or query expression for the pods)
		for _, pod := range pods.Items {
			if debug {
				fmt.Printf("Found pod for deployment %s: as %s", deploymentName, pod.Name)
			}
			serverPath := "/api/v1/query?query=%v"
			query := fmt.Sprintf(s.queryExp, pod.Name, pod.Name)

			tmp := fmt.Sprintf(serverPath, query)
			url := fmt.Sprintf("%v%v", s.prometheus, tmp)
			if debug {
				fmt.Println(url)
			}
			res, err := http.Get(url)
			if err != nil {
				glog.Fatal(err)
			}
			jsonString, err := ioutil.ReadAll(res.Body)
			res.Body.Close()
			if err != nil {
				glog.Fatal(err)
			}
			if debug {
				fmt.Printf("%s", jsonString)
			}
			// when new pods come up they won't have data yet so check that we have valid data
			checkValid := string(jsonString)
			if !strings.Contains(checkValid, "{\"resultType\":\"vector\",\"result\":[{\"metric\":{},\"value\":[") {
				if debug {
					fmt.Printf("Missing cpu data for pod: %s\n", pod.Name)
					fmt.Printf("received data: %s\n", checkValid)
				}
				continue
			}
			//get cpu usage from response
			metric := gjson.GetManyBytes(jsonString, "data.result.0.value")[0].Array()[1]
			cpu, err := strconv.ParseFloat(metric.Str, 64)
			if err != nil {
				glog.Fatal(err)
			}

			podCpuUsages = append(podCpuUsages,cpu)
		}

		//average cpu usage across all pods
		var total float64 = 0
		var podsWithData float64 = 0
		for _, cpuUsage := range podCpuUsages {
			if cpuUsage > 0 {
				total += cpuUsage
				podsWithData++
			}
		}
		averageCpuUsage := total / podsWithData

		hpaEntity := HpaEntity {
			MaxReplicas: hpa.Spec.MaxReplicas,
			MinReplicas: *hpa.Spec.MinReplicas,
			TargetCPU: *hpa.Spec.TargetCPUUtilizationPercentage,
			CurrentCPU: averageCpuUsage,
			Deployment: deployment,
			CurrentReplicas: *deployment.Spec.Replicas,
		}
		hpaEntities = append(hpaEntities, hpaEntity)
	}
	s.HpaEntities = hpaEntities
}

func (s *AutoScaler) pollCpuUsage() {

	s.generateHpaEntities()
	debug := s.debug

	//fmt.Printf("current cpu: %v\n", s.HpaEntities[0].CurrentCPU)
	//fmt.Printf("Difference: %v\n", s.HpaEntities[0].CurrentCPU/float64(s.HpaEntities[0].TargetCPU))

	// for each hpa check if needs scale
	for idx, _ := range s.HpaEntities {
		targetCpu := float64(s.HpaEntities[idx].TargetCPU)
		// check scale up?
		if s.HpaEntities[idx].CurrentCPU > targetCpu {
			if s.HpaEntities[idx].MaxReplicas == s.HpaEntities[idx].CurrentReplicas {
				continue
			}

			difference := s.HpaEntities[idx].CurrentCPU / targetCpu
			if debug {
				fmt.Printf("difference from target %s = %v \n", s.HpaEntities[idx].Deployment, difference)
			}

			if difference > s.ScaleUpThreshold {
				multiplier := 1 + ((difference - 1) * s.ScaleUpConstant)
				desiredReplicas := math.Ceil(multiplier * float64(s.HpaEntities[idx].CurrentReplicas))
				if desiredReplicas > float64(s.HpaEntities[idx].MaxReplicas) {
					desiredReplicas = float64(s.HpaEntities[idx].MaxReplicas)
				}
				replicasToSet := int32(desiredReplicas)
				s.HpaEntities[idx].Deployment.Spec.Replicas = &replicasToSet

				fmt.Printf("Increasing deployment %s to replicas %d \n", s.HpaEntities[idx].Deployment.Name, replicasToSet)

				returnDeployment, err := s.clientset.Deployments(s.Namespace).Update(s.HpaEntities[idx].Deployment)
				if err != nil {
					glog.Fatal(err)
				}

				s.recentlyScaled = s.ScaleDelay
				s.HpaEntities[idx].CurrentReplicas = *returnDeployment.Spec.Replicas
			}
		}

		//check scale down?
		if s.HpaEntities[idx].CurrentCPU < targetCpu {
			if s.HpaEntities[idx].MinReplicas == s.HpaEntities[idx].CurrentReplicas {
				continue
			}

			difference := s.HpaEntities[idx].CurrentCPU / targetCpu
			if debug {
				fmt.Printf("difference from target %s = %v \n", s.HpaEntities[idx].Deployment, difference)
			}

			//if difference < s.ScaleDownThreshold  {
			if difference < 1  {
				multiplier := 1 - ((1 - difference) * s.ScaleDownConstant)
				desiredReplicas := math.Floor(multiplier * float64(s.HpaEntities[idx].CurrentReplicas))
				fmt.Printf("desired replicas %v\n", desiredReplicas)
				if desiredReplicas < float64(s.HpaEntities[idx].MinReplicas) {
					desiredReplicas = float64(s.HpaEntities[idx].MinReplicas)
				}
				replicasToSet := int32(desiredReplicas)
				s.HpaEntities[idx].Deployment.Spec.Replicas = &replicasToSet

				fmt.Printf("Decreasing deployment %s to replicas %d \n", s.HpaEntities[idx].Deployment.Name, replicasToSet)

				returnDeployment, err := s.clientset.Deployments(s.Namespace).Update(s.HpaEntities[idx].Deployment)
				if err != nil {
					glog.Fatal(err)
				}

				s.recentlyScaled = s.ScaleDelay
				s.HpaEntities[idx].CurrentReplicas = *returnDeployment.Spec.Replicas
			}
		}
	}
}
