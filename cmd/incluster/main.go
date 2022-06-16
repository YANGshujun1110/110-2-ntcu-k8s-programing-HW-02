package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	namespace = "default"
)

func main() {
	outsideCluster := flag.Bool("outside-cluster", false, "set to true when run out of cluster. (default: false)")
	flag.Parse()

	var clientset *kubernetes.Clientset
	if *outsideCluster {
		// creates the out-cluster config
		home, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		config, err := clientcmd.BuildConfigFromFlags("", path.Join(home, ".kube/config"))
		if err != nil {
			panic(err.Error())
		}
		// creates the clientset
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}
	} else {
		// creates the in-cluster config
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
		// creates the clientset
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}
	}

	dm := createDeployment(clientset)
	sm := createService(clientset)

	go func() {
		for{
		read, err := clientset.
			AppsV1().
			Deployments(namespace).
			Get(
				context.Background(),
				dm.GetName(),
				metav1.GetOptions{},
			)
		if err != nil {
			panic(err.Error())
		}

		fmt.Printf("Read Deployment %s/%s\n", namespace, read.GetName())
		time.Sleep(time.Second)
		}
	}()

	fmt.Println("Waiting for Kill Signal...")
	var stopChan = make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-stopChan

	fmt.Printf("Delete Deployment %s/%s ", namespace, dm.GetName())
	deleteDeployment(clientset, dm)
	fmt.Printf("Delete Service %s/%s ", namespace, sm.GetName())
	deleteService(clientset, sm)
}

func int32Ptr(i int32) *int32 { return &i }

func createDeployment(client kubernetes.Interface) *appv1.Deployment {
	dm := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "orange-app1",
			Labels: map[string]string{
				"ntcu-k8s": "hw2",
			},
		},
		Spec: appv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"ntcu-k8s": "hw2",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"ntcu-k8s": "hw2",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx-container",
							Image: "nginx:1.14.2",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 80,
									Protocol:      corev1.ProtocolTCP,
								},
							},
						},
					},
				},
			},
		},
	}
	dm.Namespace = namespace

	dm, err := client.
		AppsV1().
		Deployments(namespace).
		Create(
			context.Background(),
			dm,
			metav1.CreateOptions{},
		)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("Created Deployment %s/%s\n", dm.GetNamespace(), dm.GetName())
	return dm
}

func deleteDeployment(client kubernetes.Interface, dm *appv1.Deployment) {
	err := client.
		AppsV1().
		Deployments(dm.GetNamespace()).
		Delete(
			context.Background(),
			dm.GetName(),
			metav1.DeleteOptions{},
		)
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Deleted Deployment %s/%s\n", dm.GetNamespace(), dm.GetName())
}

func deleteService(client kubernetes.Interface, sm *corev1.Service) {
	err := client.
		CoreV1().
		Services(sm.GetNamespace()).
		Delete(
			context.Background(),
			sm.GetName(),
			metav1.DeleteOptions{},
		)
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Deleted Service %s/%s\n", sm.GetNamespace(), sm.GetName())
}

var portnum int32 = 80

func createService(client kubernetes.Interface) *corev1.Service {
	sm := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "orange-service",
			Labels: map[string]string{
				"ntcu-k8s": "hw2",
		},
},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"ntcu-k8s": "hw2",
			},
			Type: corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.IntOrString{IntVal: portnum},
					NodePort:   30100,
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
	sm.Namespace = namespace
	sm, err := client.
		CoreV1().
		Services(namespace).Create(
		context.Background(),
		sm,
		metav1.CreateOptions{},
	)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("Created Deplyment %s/%s\n", sm.GetNamespace(), sm.GetName())
	return sm
}
