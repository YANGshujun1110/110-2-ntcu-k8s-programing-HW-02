package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"flag"

	"os/signal"
	"syscall"
	"time"

	appv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	namespace       = "default"
	replicas  int32 = 1

	deployment = &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nginx",
			Labels: map[string]string{
				"app":      "nginx",
				"ntcu-k8s": "hw2",
			},
		},
		Spec: appv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":      "nginx",
					"ntcu-k8s": "hw2",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nginx",
					Labels: map[string]string{
						"app":      "nginx",
						"ntcu-k8s": "hw2",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.16.1",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	service = &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nginx-svc",
			Labels: map[string]string{
				"app":      "nginx",
				"ntcu-k8s": "hw2",
			},
		},
		Spec: apiv1.ServiceSpec{
			Selector: map[string]string{
				"app": "nginx",
			},
			Type: corev1.ServiceTypeNodePort,
			Ports: []apiv1.ServicePort{
				{
					Name:     "http",
					Port:     80,
					NodePort: 30101,
					Protocol: apiv1.ProtocolTCP,
				},
			},
		},
	}
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

	clientset.AppsV1().Deployments(namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})

	clientset.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})

	cm := createConfigMap(clientset)

	go func() {
		for {
			readConfigMap(clientset, cm.GetName())
			time.Sleep(5 * time.Second)
		}
	}()

	var stopChan = make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-stopChan

	deleteConfigMap(clientset, cm)

	clientset.AppsV1().Deployments(namespace).Delete(context.TODO(), deployment.ObjectMeta.Name, metav1.DeleteOptions{})
	clientset.CoreV1().Services(namespace).Delete(context.TODO(), service.ObjectMeta.Name, metav1.DeleteOptions{})
}

func createConfigMap(client kubernetes.Interface) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{Data: map[string]string{"foo": "bar"}}
	cm.Namespace = namespace
	cm.GenerateName = "nginx"

	cm, err := client.
		CoreV1().
		ConfigMaps(namespace).
		Create(
			context.Background(),
			cm,
			metav1.CreateOptions{},
		)
	if err != nil {
		panic(err.Error())
	}

	return cm
}

func readConfigMap(clientset kubernetes.Interface, name string) *corev1.ConfigMap {
	read, err := clientset.
		CoreV1().
		ConfigMaps(namespace).
		Get(
			context.Background(),
			name,
			metav1.GetOptions{},
		)
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("%s\n", deployment.ObjectMeta.Name)
	return read
}

func deleteConfigMap(client kubernetes.Interface, cm *corev1.ConfigMap) {
	err := client.
		CoreV1().
		ConfigMaps(cm.GetNamespace()).
		Delete(
			context.Background(),
			cm.GetName(),
			metav1.DeleteOptions{},
		)
	if err != nil {
		panic(err.Error())
	}
}
