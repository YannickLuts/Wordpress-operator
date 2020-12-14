# Wordpress-operator

This wordpress-operator aims to provide a simple way of deploying a wordpress environment inside your Kubernetes cluster.


# Goals
The goals of this operator are:

 - Create a scaleable, highly available, mysql-deployment connected to the wordpress
 - Create a NFS to share the worpress files over all deployments
 - Create a wordpress deployment which can scale into different tiers
 - A preinstalled wordpress

## Installation
Currently the installation is done by downloading the source code and using the Makefile to deploy the operator locally.

```bash
git clone https://github.com/YannickLuts/Wordpress-operator
```
When the repo has been cloned, you can install the Operator into you cluster.
This command will install the CRDs into the Kubernetes cluster:
```bash
make install
```
This command will run the Kubernetes operator locally.
```bash
make run
```
## Create a Wordpress resource
Creating a custom Wordpress resource is very simple.

```yaml
apiVersion: wp.gluo.be/v1alpha1
kind: Wordpress
metadata:
  name: staging-sample
spec:
  state: "Active" #State determines weither or not the current operator should run resources -- If archived, every resources is created but replicas will be set to 0
  size: "Small" #Size specifies the resources for each deployment -- Small, Medium, Large
  tier: "Bronze" #This helps specify the replicas -- Bronze, Silver, Gold
  wordpressInfo:
    title: "Stagingphase"
    url: "wordpress-cluster.google.gluo.cloud"
```