/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.certifica
*/

package controllers

import (
	"context"
	"fmt"
	"strings"

	passGen "github.com/sethvargo/go-password/password"

	acme "github.com/jetstack/cert-manager/pkg/apis/acme/v1"
	cert "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	certmetav1 "github.com/jetstack/cert-manager/pkg/apis/meta/v1"

	networkingv1 "k8s.io/api/networking/v1beta1"

	appsv1 "k8s.io/api/apps/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	corev1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"gluo.be/api/v1alpha1"
	wpv1alpha1 "gluo.be/api/v1alpha1"
)

// WordpressReconciler reconciles a Wordpress object
type WordpressReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var reconciles = 0
var wpPassword, wpPasswordError = passGen.Generate(10, 4, 0, false, false)

// +kubebuilder:rbac:groups=wp.gluo.be,resources=wordpresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wp.gluo.be,resources=wordpresses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=statefulsets;deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=issuers;certificates;clusterissuers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services;configmaps;secrets;persistentvolumes;persistentvolumeclaims;pods,verbs=*
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses;ingressclasses,verbs=create;list;update;patch;get;delete;watch
// +kubebuilder:rbac:groups=apiregistration.k8s.io,resources=apiservices,verbs=get;list;watch;update

func (r *WordpressReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("wordpress", req.NamespacedName)

	// INFO Show the amount of times the operator reconciled
	log.Info("==Amount of Reconciles==")
	fmt.Println(reconciles)
	log.Info("============")

	log.Info("Generated PAssword: " + wpPassword)

	// ? Fetch the instance
	wordpress := &wpv1alpha1.Wordpress{}
	err := r.Get(ctx, req.NamespacedName, wordpress)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// ? Create a ConfigMap for the MySQL DB
	mconf := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: wordpress.Name + "-mysql-configmap", Namespace: wordpress.Namespace}, mconf)
	if err != nil && errors.IsNotFound(err) {
		configmap := r.CreateMySQLConfigMap(wordpress)
		log.Info("Creeren van nieuwe MySQL ConfigMap gestart")
		err = r.Create(ctx, configmap)
		if err != nil {
			log.Error(err, "Kon de configmap niet aanmaken")
			return ctrl.Result{}, err
		}
		log.Info("MySQL ConfigMap aangemaakt... Requeue")
		return ctrl.Result{Requeue: false}, nil
	} else if err != nil {
		log.Error(err, "Kon de MySQL ConfigMap niet vinden")
		return ctrl.Result{}, err
	}

	// ? Create a headless service for the MySQL DB, for stable DNS entries of StatefulSet Members.
	mysqlHeadlessService := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: wordpress.Name + "-mysql-service", Namespace: wordpress.Namespace}, mysqlHeadlessService)
	if err != nil && errors.IsNotFound(err) {
		svcHeadless := r.CreateHeadlessService(wordpress)
		log.Info("Creëren van een nieuwe MySQL Headless Service gestart")
		err = r.Create(ctx, svcHeadless)
		if err != nil {
			log.Error(err, "Kon de MySQL Headless Service niet aanmaken...")
			return ctrl.Result{}, err
		}
		log.Info("MySQL Headless Service aangemaakt... Requeue")
		return ctrl.Result{Requeue: false}, nil

	} else if err != nil {
		log.Error(err, "Kon de MySQL Headless Service niet vinden")
		return ctrl.Result{}, err
	}

	// ? Create the Reader Service.
	mysqlReaderService := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: wordpress.Name + "-mysql-read-service", Namespace: wordpress.Namespace}, mysqlReaderService)
	if err != nil && errors.IsNotFound(err) {
		svc := r.CreateReadService(wordpress)
		log.Info("Creëren van een nieuwe MySQL Reader Service gestart")
		err = r.Create(ctx, svc)
		if err != nil {
			log.Error(err, "Kon de MySQL Reader Service niet aanmaken...")
			return ctrl.Result{}, err
		}
		log.Info("MySQL Reader Service aangemaakt... Requeue")
		return ctrl.Result{Requeue: false}, nil
	} else if err != nil {
		log.Error(err, "Kon de MySQL Reader Service niet vinden")
		return ctrl.Result{}, err
	}

	// ? Create the MySQL StatefulSet
	mysqlStatefulSet := &appsv1.StatefulSet{}
	err = r.Get(ctx, types.NamespacedName{Name: wordpress.Name + "-mysql", Namespace: wordpress.Namespace}, mysqlStatefulSet)
	if err != nil && errors.IsNotFound(err) {
		statefulSet := r.CreateMySQLStatefulSet(wordpress, mysqlHeadlessService, mconf)
		log.Info("Creëren van een nieuwe MySQL StatefulSet gestart")
		err = r.Create(ctx, statefulSet)
		if err != nil {
			log.Error(err, "Couldn't create the MySQL StatefulSet")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: false}, nil
	} else if err != nil {
		log.Error(err, "Couldn't find the MySQL StatefulSet")
		return ctrl.Result{}, err
	}

	//TODO: Make sure that the wordpress deletes first before deleting the NFS
	// ? Create the NFS Service
	nfsService := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: wordpress.Name + "-nfs-server", Namespace: wordpress.Namespace}, nfsService)
	if err != nil && errors.IsNotFound(err) {
		service := r.CreateNFSservice(wordpress)
		log.Info("Creating the NFS Service now...")
		err = r.Create(ctx, service)
		if err != nil {
			log.Error(err, "Couldn't create the NFS Service")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: false}, nil
	} else if err != nil {
		log.Error(err, "Couldn't find the NFS Service")
		return ctrl.Result{}, err
	}

	// ? Create the PersistentVolumeClaim and PersistentVolume for the NFS
	nfsPVC := &corev1.PersistentVolumeClaim{}
	err = r.Get(ctx, types.NamespacedName{Name: wordpress.Name + "-nfs-pvc", Namespace: wordpress.Namespace}, nfsPVC)
	if err != nil && errors.IsNotFound(err) {
		pvc := r.CreateNFSPersistentVolumeClaim(wordpress)
		pv := r.CreateNFSPersistentVolume(wordpress)
		log.Info("Creating the NFS PVC now...")
		err = r.Create(ctx, pvc)
		if err != nil {
			log.Error(err, "Couldn't create the Persistent Volume Claim for the NFS")
			return ctrl.Result{}, err
		}
		err = r.Create(ctx, pv)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: false}, nil
	} else if err != nil {
		log.Error(err, "Couldn't find the NFS Persistent Volume Claim")
		return ctrl.Result{}, err
	}

	// ? Create the NFS Deployment
	nfsDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: wordpress.Name + "-nfs-deployment", Namespace: wordpress.Namespace}, nfsDeployment)
	if err != nil && errors.IsNotFound(err) {
		dep := r.CreateNFSdeployment(wordpress)
		log.Info("Creating the NFS Deployment")
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Couldn't create the NFS Deployment")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "Couldn't find the NFS Deployment ")
		return ctrl.Result{}, err
	}

	// ? Create the wordpress Service
	wordpressService := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: wordpress.Name + "-service", Namespace: wordpress.Namespace}, wordpressService)
	if err != nil && errors.IsNotFound(err) {
		svc := r.CreateWordpressService(wordpress)
		log.Info("Creëren van een nieuwe Wordpress Service gestart")
		err = r.Create(ctx, svc)
		if err != nil {
			log.Error(err, "Couldn't Create the Wordpress Service")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: false}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// ? Create the issuer to make SSL possible
	wordpressIssuer := &cert.Issuer{}
	err = r.Get(ctx, types.NamespacedName{Name: wordpress.Name + "-issuer", Namespace: wordpress.Namespace}, wordpressIssuer)
	if err != nil && errors.IsNotFound(err) {
		issuer := r.CreateIssuer(wordpress)
		log.Info("Creating the Wordpress Issuer to get a certificate")
		err = r.Create(ctx, issuer)
		if err != nil {
			log.Error(err, "Couldn't create the Issuer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: false}, nil

	} else if err != nil {
		log.Error(err, "Couldn't find the Issuer")
		return ctrl.Result{}, err
	}
	// ? Create the ingress for the wordpress
	wordpressIngress := &networkingv1.Ingress{}
	err = r.Get(ctx, types.NamespacedName{Name: wordpress.Name + "-ingress", Namespace: wordpress.Namespace}, wordpressIngress)
	if err != nil && errors.IsNotFound(err) {
		ingress := r.CreateIngress(wordpress, wordpressService, wordpressIssuer)
		log.Info("Creëren van een nieuwe Ingress gestart")
		err = r.Create(ctx, ingress)
		if err != nil {
			log.Error(err, "Couldn't create the Ingress")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: false}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// ? create the Wordpress Deployment
	wordpressDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: wordpress.Name + "-dep", Namespace: wordpress.Namespace}, wordpressDeployment)
	if err != nil && errors.IsNotFound(err) {
		dep := r.CreateWordpress(wordpress, mysqlHeadlessService, nfsPVC)
		log.Info("Creëren van een nieuwe Wordpress Deployment gestart")
		err = r.Create(ctx, dep)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: false}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// ? Stap 8: Lets check what the requirements of the operator are, Like State, Size and Tier
	//TODO FIX THE RESOURCES (Ofwel hier minder requirements, ofwel meer resources in le cloud)
	state := strings.ToLower(wordpress.Spec.State)
	switch state {
	case "active":
		log.Info("State is currently: ACTIVE")
		size := strings.ToLower(wordpress.Spec.Size)
		switch size {
		case "small":
			log.Info("Size is set to Small")
			mysqlStatefulSet.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{ // Change the Resources for the MySQL Statefulset
				Requests: corev1.ResourceList{"cpu": resource.MustParse("0.2"), "memory": resource.MustParse("200Mi")},
				Limits:   corev1.ResourceList{"cpu": resource.MustParse("1"), "memory": resource.MustParse("1Gi")},
			}
			r.Update(ctx, mysqlStatefulSet)                                                               // Update the MySQL Statefulset
			wordpressDeployment.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{ // Change the Resources for the Wordpress Deployment
				Requests: corev1.ResourceList{"cpu": resource.MustParse("0.1"), "memory": resource.MustParse("100Mi")},
				Limits:   corev1.ResourceList{"cpu": resource.MustParse("0.5"), "memory": resource.MustParse("512Mi")},
			}
			r.Update(ctx, wordpressDeployment) // Update the wordpress Deployment
		case "medium":
			log.Info("Size is set to Medium")
			mysqlStatefulSet.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
				Requests: corev1.ResourceList{"cpu": resource.MustParse("1"), "memory": resource.MustParse("1Gi")},
				Limits:   corev1.ResourceList{"cpu": resource.MustParse("1"), "memory": resource.MustParse("1Gi")},
			}
			r.Update(ctx, mysqlStatefulSet)
			wordpressDeployment.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
				Requests: corev1.ResourceList{"cpu": resource.MustParse("1"), "memory": resource.MustParse("1Gi")},
				Limits:   corev1.ResourceList{"cpu": resource.MustParse("1"), "memory": resource.MustParse("1Gi")},
			}
			r.Update(ctx, wordpressDeployment)
		case "large":
			log.Info("Size is set to Large")
			mysqlStatefulSet.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
				Requests: corev1.ResourceList{"cpu": resource.MustParse("1"), "memory": resource.MustParse("1Gi")},
				Limits:   corev1.ResourceList{"cpu": resource.MustParse("1"), "memory": resource.MustParse("1Gi")},
			}
			r.Update(ctx, mysqlStatefulSet)
			wordpressDeployment.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
				Requests: corev1.ResourceList{"cpu": resource.MustParse("2"), "memory": resource.MustParse("2Gi")},
				Limits:   corev1.ResourceList{"cpu": resource.MustParse("2"), "memory": resource.MustParse("2Gi")},
			}
			r.Update(ctx, wordpressDeployment)
		}

		tier := strings.ToLower(wordpress.Spec.Tier)
		switch tier {
		case "bronze":
			log.Info("Tier is currently set to Bronze")
			*mysqlStatefulSet.Spec.Replicas = 1
			r.Update(ctx, mysqlStatefulSet)
			if mysqlStatefulSet.Status.ReadyReplicas == *mysqlStatefulSet.Spec.Replicas {
				*wordpressDeployment.Spec.Replicas = 1
				r.Update(ctx, wordpressDeployment)
			}
		case "silver":
			log.Info("Tier is currently set to Silver")
			*mysqlStatefulSet.Spec.Replicas = 1
			r.Update(ctx, mysqlStatefulSet)
			if mysqlStatefulSet.Status.ReadyReplicas == *mysqlStatefulSet.Spec.Replicas {
				*wordpressDeployment.Spec.Replicas = 2
				r.Update(ctx, wordpressDeployment)
			}
		case "gold":
			log.Info("Tier is currently set to Gold")
			*mysqlStatefulSet.Spec.Replicas = 2
			r.Update(ctx, mysqlStatefulSet)
			if mysqlStatefulSet.Status.ReadyReplicas == 1 { //== *mysqlStatefulSet.Spec.Replicas
				*wordpressDeployment.Spec.Replicas = 3
				r.Update(ctx, wordpressDeployment)
			}

		}
	case "archived":
		log.Info("State is currently: ARCHIVED")
		*mysqlStatefulSet.Spec.Replicas = 0
		r.Update(ctx, mysqlStatefulSet)
		*wordpressDeployment.Spec.Replicas = 0
		r.Update(ctx, wordpressDeployment)
	}

	// Update the URL status
	wordpress.Status.URL = "https://" + wordpressIngress.Spec.Rules[0].Host
	err = r.Status().Update(ctx, wordpress)
	if err != nil {
		return ctrl.Result{}, err
	} else {
		log.Info("STATUS UPDATE: URL")
	}

	// Update the Wordpress resource status to display the generated password
	if wordpress.Status.Password == "" || wordpress.Status.Password != wpPassword {
		wordpress.Status.Password = wpPassword
		err = r.Status().Update(ctx, wordpress)
		if err != nil {
			log.Error(err, "Something went wrong updating the status of the password")
			return ctrl.Result{}, err
		}
	}

	reconciles++
	return ctrl.Result{}, nil
}

// SetupWithManager ...
func (r *WordpressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wpv1alpha1.Wordpress{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolume{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&cert.Issuer{}).
		Owns(&cert.ClusterIssuer{}).
		Owns(&cert.Certificate{}).
		Complete(r)
}

// ! Functies die Objecten creëren

// CreateMySQLConfigMap ...
func (r *WordpressReconciler) CreateMySQLConfigMap(w *wpv1alpha1.Wordpress) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      w.Name + "-mysql-configmap",
			Namespace: w.Namespace,
			Labels:    map[string]string{"app": "mysql"},
		}, Data: map[string]string{"master.cnf": "[mysqld]\nlog-bin", "slave.cnf": "[mysqld]\nsuper-read-only"}, //Schrijffout bij slave -> ik had eerst Slafe *INsert facepalm*
	}
	ctrl.SetControllerReference(w, cm, r.Scheme)
	return cm
}

// CreateHeadlessService ...
func (r *WordpressReconciler) CreateHeadlessService(w *wpv1alpha1.Wordpress) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      w.Name + "-mysql-service",
			Namespace: w.Namespace,
			Labels:    map[string]string{"app": "mysql"},
		}, Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "mysql"},
			Ports: []corev1.ServicePort{{
				Port: 3306,
			}},
			ClusterIP: "None",
		},
	}
	ctrl.SetControllerReference(w, svc, r.Scheme)
	return svc
}

// CreateReadService ...
func (r *WordpressReconciler) CreateReadService(w *wpv1alpha1.Wordpress) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      w.Name + "-mysql-read-service",
			Namespace: w.Namespace,
			Labels:    map[string]string{"app": "mysql"},
		}, Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "mysql"},
			Ports: []corev1.ServicePort{{
				Port: 3306,
			}},
		},
	}
	ctrl.SetControllerReference(w, svc, r.Scheme)
	return svc
}

// CreateMySQLStatefulSet ...
func (r *WordpressReconciler) CreateMySQLStatefulSet(w *wpv1alpha1.Wordpress, s *corev1.Service, c *corev1.ConfigMap) *appsv1.StatefulSet {

	name := string(w.Name + "-mysql")
	replicas := int32(0)

	commandInitContainer := `set -ex` +
		"\n [[ `hostname` =~ -([0-9]+)$ ]] || exit 1\n" +
		`ordinal=${BASH_REMATCH[1]}
	echo [mysqld] > /mnt/conf.d/server-id.cnf
	# Add an offset to avoid reserved server-id=0 value.
	echo server-id=$((100 + $ordinal)) >> /mnt/conf.d/server-id.cnf
	# Copy appropriate conf.d files from config-map to emptyDir.
	if [[ $ordinal -eq 0 ]]; then
	  cp /mnt/config-map/master.cnf /mnt/conf.d/
	else
	  cp /mnt/config-map/slave.cnf /mnt/conf.d/
	fi`

	commandCloneContainer := `set -ex
	[[ -d /var/lib/mysql/mysql ]] && exit 0` +
		"\n[[ `hostname` =~ -([0-9]+)$ ]] || exit 1\n" +
		`ordinal=${BASH_REMATCH[1]}
	[[ $ordinal -eq 0 ]] && exit 0
	ncat --recv-only ` + name + `-$(($ordinal-1)).` + s.Name + ` 3307 | xbstream -x -C /var/lib/mysql
	xtrabackup --prepare --target-dir=/var/lib/mysql`

	commandXtraBackup := `
set -ex
cd /var/lib/mysql

if [[ -f xtrabackup_slave_info && "x$(<xtrabackup_slave_info)" != "x" ]]; then
cat xtrabackup_slave_info | sed -E 's/;$//g' > change_master_to.sql.in
rm -f xtrabackup_slave_info xtrabackup_binlog_info
elif [[ -f xtrabackup_binlog_info ]]; then` +
		"\n[[ `cat xtrabackup_binlog_info` =~ ^(.*?)[[:space:]]+(.*?)$ ]] || exit 1\n" + //Hier zit nen error
		`rm -f xtrabackup_binlog_info xtrabackup_slave_info
    echo "CHANGE MASTER TO MASTER_LOG_FILE='${BASH_REMATCH[1]}',\
    	MASTER_LOG_POS=${BASH_REMATCH[2]}" > change_master_to.sql.in
    fi

    if [[ -f change_master_to.sql.in ]]; then
        echo "Waiting for mysqld to be ready (accepting connections)"
        until mysql -h 127.0.0.1 -e "SELECT 1"; do sleep 1; done

        echo "Initializing replication from clone position"
        mysql -h 127.0.0.1 \
        -e "$(<change_master_to.sql.in), \
            MASTER_HOST='` + name + `-0.` + s.Name + `', \
            MASTER_USER='root', \
            MASTER_PASSWORD='', \
            MASTER_CONNECT_RETRY=10; \
        START SLAVE;" || exit 1
        mv change_master_to.sql.in change_master_to.sql.orig
    fi
	exec ncat --listen --keep-open --send-only --max-conns=1 3307 -c \
    	"xtrabackup --backup --slave-info --stream=xbstream --host=127.0.0.1 --user=root"
	`

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: w.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "mysql"},
			},
			ServiceName: s.Name,
			Replicas:    &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "mysql"},
				}, Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "init-mysql",
							Image: "mysql:5.7",
							Command: []string{"bash",
								"-c",
								commandInitContainer,
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "conf", MountPath: "/mnt/conf.d"},
								{Name: "config-map", MountPath: "/mnt/config-map"}},
						},
						{
							Name:  "clone-mysql",
							Image: "gcr.io/google-samples/xtrabackup:1.0",
							Command: []string{"bash",
								"-c",
								commandCloneContainer,
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "data", MountPath: "/var/lib//mysql", SubPath: "mysql"},
								{Name: "conf", MountPath: "/etc/mysql/conf.d"},
							},
						},
					},
					// Containers
					Containers: []corev1.Container{{
						// * Mysql hoofd container
						Name:  "mysql",
						Image: "mysql:5.7",
						Env: []corev1.EnvVar{{
							Name:  "MYSQL_ALLOW_EMPTY_PASSWORD",
							Value: "1",
						}},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 3306,
						}},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{"cpu": resource.MustParse("250m"), "memory": resource.MustParse("500Mi")},
						},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{Exec: &corev1.ExecAction{
								Command: []string{"mysqladmin", "ping"},
							}},
							InitialDelaySeconds: 30,
							PeriodSeconds:       10,
							TimeoutSeconds:      5,
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{Exec: &corev1.ExecAction{
								Command: []string{"mysql", "-h", "127.0.0.1", "-e", "SELECT 1"},
							}},
							InitialDelaySeconds: 15, //mss 15
							PeriodSeconds:       2,
							TimeoutSeconds:      1,
						},
						// VolumeMounts van de hoofd Container
						VolumeMounts: []corev1.VolumeMount{
							{Name: "data", MountPath: "/var/lib/mysql", SubPath: "mysql"},
							{Name: "conf", MountPath: "/etc/mysql/conf.d"}},
					}, {
						Name:  "xtrabackup",
						Image: "gcr.io/google-samples/xtrabackup:1.0",
						Ports: []corev1.ContainerPort{{
							ContainerPort: 3307,
						}},
						Command: []string{"bash",
							"-c",
							commandXtraBackup,
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "data", MountPath: "/var/lib/mysql", SubPath: "mysql"},
							{Name: "conf", MountPath: "/etc/mysql/conf.d"},
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{"cpu": resource.MustParse("100m"), "memory": resource.MustParse("100Mi")},
						},
					},
					},
					Volumes: []corev1.Volume{
						{
							Name:         "conf",
							VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
						},
						{
							Name: "config-map",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: c.Name},
								},
							},
						},
					},
				},
			}, VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
				ObjectMeta: metav1.ObjectMeta{
					Name: "data",
				}, Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{"storage": resource.MustParse("10Gi")},
					},
				},
			}},
		},
	}
	ctrl.SetControllerReference(w, statefulSet, r.Scheme)
	return statefulSet
}

func (r *WordpressReconciler) CreateWordpressService(w *v1alpha1.Wordpress) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      w.Name + "-service",
			Namespace: w.Namespace,
			Labels:    map[string]string{"app": "wordpress"},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "wordpress"},
			Ports: []corev1.ServicePort{{
				Port:       80,
				TargetPort: intstr.IntOrString{IntVal: 80},
			}},
		},
	}
	ctrl.SetControllerReference(w, svc, r.Scheme)
	return svc
}

func (r *WordpressReconciler) CreateWordpress(w *v1alpha1.Wordpress, s *corev1.Service, p *corev1.PersistentVolumeClaim) *appsv1.Deployment {

	replicas := int32(0)
	url := w.Spec.WordpressInfo.URL
	title := w.Spec.WordpressInfo.Title
	admin := w.Spec.WordpressInfo.AdminUser
	adminPass := w.Spec.WordpressInfo.AdminPass
	adminEmail := w.Spec.WordpressInfo.AdminEmail

	// Check if the admin password is empty
	if adminPass == "" {
		adminPass = wpPassword
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      w.Name + "-dep",
			Namespace: w.Namespace,
			Labels:    map[string]string{"app": "wordpress"},
		},
		Spec: appsv1.DeploymentSpec{
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
			},
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "wordpress"},
			}, Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "wordpress"},
				}, Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "wp-container",
						Image: "conetix/wordpress-with-wp-cli",
						Env: []corev1.EnvVar{{
							Name:  "WORDPRESS_DB_HOST",
							Value: s.Name,
						}, {
							Name:  "WORDPRESS_DB_PASSWORD",
							Value: "",
						}},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 80,
						}},
						Lifecycle: &corev1.Lifecycle{
							PostStart: &corev1.Handler{
								Exec: &corev1.ExecAction{
									Command: []string{
										"bash",
										"-c",
										"wp core install --url=" + url + " --title=" + title + " --admin_user=" + admin + " --admin_email=" + adminEmail + " --admin_password=" + adminPass,
										// ? Command might not work
									},
								},
							},
						},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "workingdir",
							MountPath: "/var/www/html/",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "workingdir",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: p.Name,
							},
						},
					}},
				},
			},
		},
	}
	ctrl.SetControllerReference(w, dep, r.Scheme)
	return dep
}

func (r *WordpressReconciler) CreateIssuer(w *v1alpha1.Wordpress) *cert.Issuer {
	class := "nginx"
	issuer := &cert.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      w.Name + "-issuer",
			Namespace: w.Namespace,
		},
		Spec: cert.IssuerSpec{
			IssuerConfig: cert.IssuerConfig{
				ACME: &acme.ACMEIssuer{
					Email: "yannick.luts@hotmail.com",
					//Server: "https://acme-staging-v02.api.letsencrypt.org/directory", // INFO Staging Server, to test wether or not cert-manager is working
					Server: "https://acme-v02.api.letsencrypt.org/directory", // INFO Production Server, This has rate limits.
					PrivateKey: certmetav1.SecretKeySelector{
						LocalObjectReference: certmetav1.LocalObjectReference{
							Name: "letsencrypt-staging",
						},
					},
					Solvers: []acme.ACMEChallengeSolver{{
						HTTP01: &acme.ACMEChallengeSolverHTTP01{
							Ingress: &acme.ACMEChallengeSolverHTTP01Ingress{
								Class: &class,
							},
						},
					}},
				},
			},
		},
	}
	ctrl.SetControllerReference(w, issuer, r.Scheme)
	return issuer
}

func (r *WordpressReconciler) CreateIngress(w *v1alpha1.Wordpress, s *corev1.Service, i *cert.Issuer) *networkingv1.Ingress {
	pathType := networkingv1.PathType(networkingv1.PathTypePrefix)
	host := w.Spec.WordpressInfo.URL
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      w.Name + "-ingress",
			Namespace: w.Namespace,
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "nginx",
				//"nginx.ingress.kubernetes.io/force-ssl-redirect": "false",
				"acme.cert-manager.io/http01-edit-in-place": "true",
				"cert-manager.io/issuer":                    i.Name,
			},
		},
		Spec: networkingv1.IngressSpec{
			TLS: []networkingv1.IngressTLS{
				{
					SecretName: "wordpress-cluster-tls",
					Hosts:      []string{host},
				},
			},
			Rules: []networkingv1.IngressRule{
				{
					Host: host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										ServiceName: s.Name,
										ServicePort: intstr.IntOrString{IntVal: s.Spec.Ports[0].Port},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ctrl.SetControllerReference(w, ing, r.Scheme)
	return ing
}

func (r *WordpressReconciler) CreateCert(w *v1alpha1.Wordpress, i *cert.Issuer) *cert.Certificate {
	cert := &cert.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      w.Name + "-cert",
			Namespace: w.Namespace,
		},
		Spec: cert.CertificateSpec{
			SecretName: "wordpress-cluster-tls",
			IssuerRef: certmetav1.ObjectReference{
				Name: i.Name,
			},
			CommonName: "wordpress-cluster.google.gluo.cloud",
			DNSNames:   []string{"wordpress-cluster.google.cluo.cloud"},
		},
	}
	ctrl.SetControllerReference(w, cert, r.Scheme)
	return cert
}

// TODO implement a NFS server, service, pv and pvc
// CreateNFSservice is a function that will create the service for the NFS
func (r *WordpressReconciler) CreateNFSservice(w *v1alpha1.Wordpress) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      w.Name + "-nfs-server",
			Namespace: w.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "nfs-server"},
			Ports: []corev1.ServicePort{
				{
					Name: "nfs",
					Port: 2049,
				},
				{
					Name: "mountd",
					Port: 20048,
				},
				{
					Name: "rcpbind",
					Port: 111,
				},
			},
		},
	}
	ctrl.SetControllerReference(w, svc, r.Scheme)
	return svc
}

// CreateNFSPersistentVolume is a function that creates the pv for the nfs
func (r *WordpressReconciler) CreateNFSPersistentVolume(w *v1alpha1.Wordpress) *corev1.PersistentVolume {
	nfsPV := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      w.Name + "-persistentvolume",
			Namespace: w.Namespace,
			Labels:    map[string]string{"app": "nfs-server"},
		}, Spec: corev1.PersistentVolumeSpec{
			Capacity:         corev1.ResourceList{"storage": resource.MustParse("20Gi")},
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			StorageClassName: "",
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				NFS: &corev1.NFSVolumeSource{
					Server: w.Name + "-nfs-server.default.svc.cluster.local",
					Path:   "/",
				},
			},
		},
	}
	ctrl.SetControllerReference(w, nfsPV, r.Scheme)
	return nfsPV
}

// CreateNFSPersistentVolumeClaim is a function that create the pvc for the pv of the nfs
func (r *WordpressReconciler) CreateNFSPersistentVolumeClaim(w *v1alpha1.Wordpress) *corev1.PersistentVolumeClaim {
	storageClassName := ""
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      w.Name + "-nfs-pvc",
			Namespace: w.Namespace,
			Labels:    map[string]string{"app": "nfs-server"},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
			StorageClassName: &storageClassName,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"storage": resource.MustParse("20Gi"),
				},
			},
		},
	}
	ctrl.SetControllerReference(w, pvc, r.Scheme)
	return pvc
}

func (r *WordpressReconciler) CreateNFSdeployment(w *v1alpha1.Wordpress) *appsv1.Deployment {
	replicas := int32(1)
	privileged := true
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      w.Name + "-nfs-deployment",
			Namespace: w.Namespace,
			Labels:    map[string]string{"app": "nfs-server"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "nfs-server"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "nfs-server"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "nfs-server",
						Image: "gcr.io/google_containers/volume-nfs:0.8",
						Ports: []corev1.ContainerPort{{
							Name:          "nfs",
							ContainerPort: 2049,
						}, {
							Name:          "mountd",
							ContainerPort: 20048,
						}, {
							Name:          "rcpbind",
							ContainerPort: 111,
						}},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &privileged,
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "nfs-server-exports",
								MountPath: "/exports",
							},
						},
					}},
					Volumes: []corev1.Volume{
						{
							Name: "nfs-server-exports",
							VolumeSource: corev1.VolumeSource{ // INFO VolumeSource can also be a persistentVolumeClaim. But then its only available inside the cluster.
								GCEPersistentDisk: &corev1.GCEPersistentDiskVolumeSource{ // ! Create a GCEPersistentDisk with the name nfs-server first
									PDName: "nfs-server-disk",
									FSType: "ext4",
								},
							},
						},
					},
				},
			},
		},
	}
	ctrl.SetControllerReference(w, dep, r.Scheme)
	return dep
}
