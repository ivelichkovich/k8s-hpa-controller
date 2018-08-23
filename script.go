//usr/bin/env go run $0 "$@"; exit
package main

import (
	"fmt"
	"os"
	"log"

	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"github.com/ivelichkovich/k8s-hpa-controller/autoscaler"
	"github.com/ivelichkovich/k8s-hpa-controller/options"
	"net/http"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func homePage(scaler *autoscaler.AutoScaler)  http.HandlerFunc {
	//fmt.Fprintf(w, "Welcome to the HomePage!")
	//fmt.Println("Endpoint Hit: homePage")
	return func(w http.ResponseWriter, r *http.Request) {
		if scaler.HpaEntities != nil {
			for _, hpaEntity := range scaler.HpaEntities {
				if hpaEntity.RunningFor5min {
					fmt.Fprintf(
						w,
						"APP: %s \n\t TARGETS: %v%% / %d%% \n\t MinReplicas: %d \t MaxReplicas: %d \t CurrentReplicas: %d \n\n",
						hpaEntity.Deployment.Name,
						hpaEntity.CurrentCPU,
						hpaEntity.TargetCPU,
						hpaEntity.MinReplicas,
						hpaEntity.MaxReplicas,
						hpaEntity.CurrentReplicas,
					)
				} else {
					fmt.Fprintf(
						w,
						"APP: %s \n\t No pod has been running for 5minutes so compute usage ignored \n\t TARGETS: %v%% / %d%% \n\t MinReplicas: %d \t MaxReplicas: %d \t CurrentReplicas: %d \n\n",
						hpaEntity.Deployment.Name,
						hpaEntity.CurrentCPU,
						hpaEntity.TargetCPU,
						hpaEntity.MinReplicas,
						hpaEntity.MaxReplicas,
						hpaEntity.CurrentReplicas,
					)
				}
			}
		} else {
			fmt.Fprint(w, "No HPAs Currently found")
		}

	}
}

func handleRequests(scaler *autoscaler.AutoScaler) {
	http.HandleFunc("/", homePage(scaler))
	log.Fatal(http.ListenAndServe(":8081", nil))
}


func main() {
	// if you're using this I'm assuming you're on EKS, my kubeconfig didn't work for local dev not sure if it's the SAML I use
	// or EKS heptio-authenticator but I didn't waste time figuring that out so i did kubectl proxy and made a kubeconfig
	// pointing at localhost:8001 and set that as this kubeconfig value below
	inClusterConfig := true
	kubeconfig := "path/to/your/secondary/kubeconfig/.kube/config-go"
	config := options.NewAutoScalerConfig()

	config.AddFlags(pflag.CommandLine)
	pflag.Parse()
	if err := config.ValidateFlags(); err != nil {
		glog.Errorf("%v\n", err)
		os.Exit(1)
	}

	//Create kubernetes clientset
	var err error
	var kubeClientConfig *rest.Config
	if inClusterConfig {
		// creates the in-cluster config
		kubeClientConfig, err = rest.InClusterConfig()
	} else {
		// uses the current context in kubeconfig
		kubeClientConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		glog.Fatalf("Error configuring the client: %v", err.Error())
	}
	// creates the clientset for Kubernetes
	clientset, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		glog.Fatalf("Failed to create Kubernetes client: %v", err.Error())
	}

	scaler, err := autoscaler.NewAutoScaler(config, clientset)
        if err != nil {
                glog.Errorf("%v", err)
                os.Exit(1)
        }
        // Begin autoscaling.
	go handleRequests(scaler)
	scaler.Run()

}

// Sample curl to prometheus
// curl -sS -g 'http://192.168.99.100:31951/api/v1/query_range?query=avg(container_spec_cpu_period{namespace="b2b-dev-hk",pod_name=~"b2b-web-.*"})&start='$(date +%s --date='5 minutes ago')'&end='$(date +%s)'&step=15s'

// Sample response json
// {"status":"success","data":{"resultType":"matrix","result":[{"metric":{},"values":[[1483953922,"100000"],[1483953937,"100000"],[1483953952,"100000"],[1483953967,"100000"],[1483953982,"100000"],[1483953997,"100000"],[1483954012,"100000"],[1483954027,"100000"],[1483954042,"100000"],[1483954057,"100000"],[1483954072,"100000"],[1483954087,"100000"],[1483954102,"100000"],[1483954117,"100000"],[1483954132,"100000"],[1483954147,"100000"],[1483954162,"100000"],[1483954177,"100000"],[1483954192,"100000"],[1483954207,"100000"],[1483954222,"100000"]]}]}}
