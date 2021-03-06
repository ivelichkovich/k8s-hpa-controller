# HPA-CONTROLLER
(new to go so comments appreciated)

# WARNING
Test this for yourself before relying on it in production

to run locally edit kubeconfig file location in script.go to point to a kubeconfig (i.e. ~/.kube/config-go) pointing that points at localhost:8001 and run kubectl proxy with your cluster kubeconfig
and set inClusterConfig bool in script.go to false

# Deploying
Comes with a Makefile to build for linux and a Dockerfile, build the docker image and push to your favorite registry than put that image in the helm chart and enjoy
Will deploy with an ingress, set your values appropriately

# Prereqs

Must be running prometheus that is scrapping cAdvisor. Make sure you have the container_cpu_usage_seconds_total and kube_pod_container_resource_requests_cpu_cores metrics
You can use your own custom query.
Options are in options.go.
Be sure to set your prometheus address and namespace

This will only work for one namespace, if you want to make it cluster wide contributions are welcome

# How it works

The controller will poll your existing native k8s HPAs to get min/max replicas and target cpu (if you want to use a custom query like memory it will still use the target from your hpa so if it's 50% it'll be a 50% memory target)
It will then poll your deployments that are the targets of the HPAs, make sure the deployment has a label app=SOMETHING that matches the label on the deployments pods
Then the controller grabs the pod names and queries prometheus to generate the usage metric you see in the UI
UI on port 8081

Query I'm using to find CPU percentage by default

sum(label_replace(rate(container_cpu_usage_seconds_total{pod_name=\"%s\"}[5m]),\"pod\",\"$1\",\"pod_name\",\"(.+)\"))/sum(kube_pod_container_resource_limits_cpu_cores{pod=\"%s\"})*100



# Options
see options.go for default values and available options, I'm not going to rewrite that into the readme
a few notes:
--prom-location should start with http:// or https:// (if you're running this on cluster you should probably use http://service_name.namespace_name)
--query-exp is used per pod so replace the pod name in the prom query to %s and it'll format. By default this is expecting the pod name to be formatted in twice so your query should reflect that unless you want to quickly update the code
also make sure the query has no spaces

There is no reason to poll more often than your prometheus scrape period because the data will be the same, if you want this more responsive prometheus must scrape more frequently

# Calculating scales
find default values in options.go

controller will calculate a difference value by actualMetricUsage/targetMetricUsage if that value is above (below) the scaleUpThreshold (scaleDownThreshold) it will try to scale
for scale up, controller will take difference - 1 as the base scale amount and multiply it by a constant.
For example: if your scaleUpConstant is 1, and your target cpu usage is 50% with actual usage 55% the difference will be 1.1
Your usage is 10% above your target so it will try to scale up by 10%. if your scale up constant is 2 it will try to scale up by .1 * 2 or 20%
similar functionality for scale down.


