# Wordpress-operator

**❗ Currently this operator only works on GKE ❗**

This wordpress-operator aims to provide a simple way of deploying a wordpress environment inside your Kubernetes cluster.
It was a project for my internship and now the operator is open source.


**Keep in mind: This operator creates a NFS running inside your kubernetes cluster. It will automatically link with a GCE Persistent Disk. This means that if you download the source code, you won't be able to run the operator locally without changing the `VolumeSource` within the `CreateNFSdeployment` function.**

# Goals
The goals of this operator are:

 - Create a scalable, highly available, mysql-deployment connected to the wordpress
 - Create a NFS to share the worpress files over all deployments
 - Create a wordpress deployment which can scale into different tiers
 - A preinstalled wordpress
 

# Prerequisite
Before installing, make sure that your cluster has enough resources to run the operator. Check the table below to have a clear view of what the required resources are for each size that can be specified.

## MySQL
| Size   | CPU   | Memory |
|:------:|:-----:|:------:|
| Small  | 1 CPU | 1Gb    |
| Medium | 1 CPU | 1Gb    |
| Large  | 1 CPU | 1GB    |

## Wordpress
| Size   | CPU   | Memory |
|:------:|:-----:|:------:|
| Small  | 0.5   | 512Mb  |
| Medium | 1 CPU | 1Gb    |
| Large  | 2 CPU | 2GB    |

Next you should make sure that you have the following things setup:

- GCE Persistent Disk with the name: **nfs-server-disk**
- install the nginx-ingress intro you cluster `helm install nginx-ingress nginx-stable/nginx-ingress`


## Installation
To install this Wordpress operator into your cluster slimply run the following command in your console.

```bash
kubectl apply -f https://raw.githubusercontent.com/YannickLuts/Wordpress-operator/master/installation/wordpress-operator.yaml
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
    title: "Title"
    url: "example.com"
```
## TODO
 - Make it so that the user has to input his email.
