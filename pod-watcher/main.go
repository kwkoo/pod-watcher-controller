package main

import (
	"log"
	"time"
	_ "time/tzdata"

	"github.com/kwkoo/configparser"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/sample-controller/pkg/signals"
)

func main() {
	config := struct {
		MasterURL     string `usage:"Kubernetes master URL"`
		Kubeconfig    string `usage:"Path to kubeconfig file"`
		AnnotationKey string `usage:"The annotation key that specifies the name of the ConfigMap" mandatory:"true"`
		NumWorkers    int    `usage:"Number of workqueue workers" default:"1"`
	}{}

	if err := configparser.Parse(&config); err != nil {
		log.Fatal(err)
	}

	var (
		cfg *rest.Config
		err error
	)

	if len(config.Kubeconfig) > 0 {
		if cfg, err = clientcmd.BuildConfigFromFlags(config.MasterURL, config.Kubeconfig); err != nil {
			log.Fatalf("could not initialize kube client from %s: %v", config.Kubeconfig, err)
		}
		log.Printf("using %s as kube config", config.Kubeconfig)
	}

	if cfg == nil {
		if cfg, err = rest.InClusterConfig(); err != nil {
			log.Fatalf("could not initialize kube client using in-cluster config: %v", err)
		}
		log.Print("using in-cluster kube config")
	}

	var clientset *kubernetes.Clientset
	if clientset, err = kubernetes.NewForConfig(cfg); err != nil {
		log.Fatalf("error initializing clientset: %v", err)
	}

	factory := informers.NewSharedInformerFactory(clientset, time.Second*30)

	// instantiate controller here
	stopCh := signals.SetupSignalHandler()
	controller := NewPodController(config.AnnotationKey, clientset, factory.Core().V1().Pods())

	if err := controller.InitNodeCache(); err != nil {
		log.Fatalf("could not initialize node cache: %v", err)
	}

	factory.Start(stopCh)

	if err := controller.Run(config.NumWorkers, stopCh); err != nil {
		log.Fatalf("error running controller: %v", err)
	}
}
