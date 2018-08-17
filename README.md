# HPA-CONTROLLER


to run locally edit kubeconfig file location in script.go to point to a kubeconfig (i.e. ~/.kube/config-go) pointing that points at localhost:8001 and run kubectl proxy with your cluster kubeconfig

# Deploying
Comes with a Makefile to build for linux and a Dockerfile, build the docker image and push to your favorite registry than put that image in the helm chart and enjoy
Will deploy with an ingress, set your values appropriately

# Prereqs

Must be running prometheus that is scrapping cAdvisor. Make sure you have the container_cpu_usage_seconds_total and kube_pod_container_resource_requests_cpu_cores metrics
You can use your own custom query. Options are in options.go. Be sure to set your prometheus address and your

code uses target cpu and cpu as default query, if you want to use a different query the hpa target % needs to be relevant to that query

UI on port 8081

Would love feedback on the query I'm using to find cpu usage, even for single container pods some odd values show up sometimes like over 100%

sum(label_replace(rate(container_cpu_usage_seconds_total{pod_name=\"%s\"}[5m]),\"pod\",\"$1\",\"pod_name\",\"(.+)\"))/sum(kube_pod_container_resource_requests_cpu_cores{pod=\"%s\"})*100


Comes with helm chart and Dockerfile (I threw it together quickly so excuse that it's not exactly best practice, contributions welcome)
