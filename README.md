# Wordpress-operator

<a href="https://hub.docker.com/repository/docker/yannickl/wordpress-operator"><img alt="Docker Image Version (tag latest semver)" src="https://img.shields.io/docker/v/yannickl/wordpress-operator/latest?logo=Docker&logoColor=white&style=for-the-badge"></a> 
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; ![GitHub top language](https://img.shields.io/github/languages/top/yannickluts/wordpress-operator?color=blueviolet&logo=go&style=for-the-badge)

This Wordpress operator aims to provide a simple way of deploying a wordpress environment inside your Kubernetes cluster.
It was a project for my internship and now the operator is open source.
It deploys a frontend Wordpress with a highly available MySQL and a NFS server. It also includes cert-manager to generate SSL certificates for you Wordpress website.

**❗ Currently this operator only works on GKE ❗**

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
To install this Wordpress operator into your cluster slimply run the following commands in your console.

First install the Nginx Ingress controller
```bash
helm install nginx-ingress nginx-stable/nginx-ingress
```

```bash
kubectl apply -f https://raw.githubusercontent.com/YannickLuts/Wordpress-operator/master/installation/wordpress-operator.yaml
```

## Create a Wordpress resource
Creating a custom Wordpress resource is very simple.

```yaml
apiVersion: wp.gluo.be/v1alpha1
kind: Wordpress
metadata:
  name: example
spec:
  state: "Active" #State determines weither or not the current operator should run resources -- If archived, every resources is created but replicas will be set to 0
  size: "Small" #Size specifies the resources for each deployment -- Small, Medium, Large
  tier: "Bronze" #This helps specify the replicas -- Bronze, Silver, Gold
  wordpressInfo:
    email: "example@example.com" # REQUIRED
    title: "Title" #Optional
    url: "example.com" #Optional
    user: #optional
    password: #optional -> When empty a password will be generated
```
## Uninstallation

Before uninstalling a wordpress resource or the operator please put the `state:` into `Archived` to prevent data loss.

## Todo
[ ] In archived state, scale the nfs down to 0

[x] Make the operator run in every namespace and not only the default one. (Should by changing the NFS Server to the current namespace.

[ ] Update deinstallation by just deleting the namespace the operator is running in.
