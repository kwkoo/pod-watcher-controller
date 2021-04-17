package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	coreinformer "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// this is what goes into the workqueue
type podInfo struct {
	name      string
	namespace string
}

type Controller struct {
	annotationKey string
	clientset     *kubernetes.Clientset
	podsLister    corelisters.PodLister
	podsSynced    cache.InformerSynced
	workqueue     workqueue.RateLimitingInterface
	nodeCache     map[string]string
}

func NewPodController(annotationKey string, clientset *kubernetes.Clientset, podInformer coreinformer.PodInformer) *Controller {
	c := Controller{
		annotationKey: annotationKey,
		clientset:     clientset,
		podsLister:    podInformer.Lister(),
		podsSynced:    podInformer.Informer().GetController().HasSynced,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ScheduledPods"),
		nodeCache:     make(map[string]string),
	}

	log.Print("setting up event handlers")
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			newPod := new.(*v1.Pod)
			c.processPodEvent(newPod)
		},
	})

	return &c
}

// InitNodeCache lists all nodes and populates the nodeCache with the data.
func (c *Controller) InitNodeCache() error {
	type nodeInfo struct {
		Name       string `json:"name"`
		ProviderID string `json:"providerid"`
		Hostname   string `json:"hostname"`
		InternalIP string `json:"internalip"`
	}
	nodes, err := c.clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("could not list nodes: %w", err)
	}
	for _, node := range nodes.Items {
		n := nodeInfo{
			Name:       node.GetObjectMeta().GetName(),
			ProviderID: node.Spec.ProviderID,
		}
		for _, a := range node.Status.Addresses {
			if a.Type == v1.NodeHostName {
				n.Hostname = a.Address
			} else if a.Type == v1.NodeInternalIP {
				n.InternalIP = a.Address
			}
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(&n)
		c.nodeCache[n.Name] = b.String()
	}

	log.Printf("node cache initialized with %d entries", len(c.nodeCache))
	return nil
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer c.workqueue.ShutDown()

	log.Print("starting controller")

	log.Print("waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.podsSynced); !ok {
		return errors.New("failed to wait for caches to sync")
	}

	log.Print("starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Print("started workers")
	<-stopCh
	log.Print("shutting down workers")
	return nil
}

func (c *Controller) processPodEvent(p *v1.Pod) {
	// filter for the right type of event
	// if we find the right type, put it on the work queue
	if len(p.Status.Conditions) == 0 || p.Status.Conditions[0].Type != "PodScheduled" {
		return
	}
	if _, ok := p.ObjectMeta.GetAnnotations()[c.annotationKey]; !ok {
		return
	}
	c.workqueue.Add(podInfo{
		name:      p.ObjectMeta.Name,
		namespace: p.ObjectMeta.Namespace,
	})
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)

		// we really should only call c.workqueue.Forget(p) after we have
		// successfully processed this - but we're calling it now because we
		// know there aren't going to be any errors during processing

		podinfo, ok := obj.(podInfo)
		if !ok {
			log.Printf("expected to pull podinfo struct off scheduled queue but got %T instead", obj)
			c.workqueue.Forget(obj)
			return nil
		}

		p, err := c.podsLister.Pods(podinfo.namespace).Get(podinfo.name)
		if err != nil {
			c.workqueue.AddRateLimited(obj)
			return fmt.Errorf("could not get pod %s in namespace %s from pod informer: %w", podinfo.name, podinfo.namespace, err)
		}

		annotationValue, ok := p.ObjectMeta.Annotations[c.annotationKey]
		if !ok {
			log.Print("this pod does not have the expected annotation")
			c.workqueue.Forget(obj)
			return nil
		}

		nodeName := p.Spec.NodeName
		if len(nodeName) == 0 {
			log.Print("this pod is not assigned to a node yet")
			c.workqueue.Forget(obj)
			return nil
		}

		namespace := p.ObjectMeta.Namespace

		log.Printf("name=%s namespace=%s nodeName=%s annotation-value=%s", p.ObjectMeta.Name, namespace, nodeName, annotationValue)

		providerid := c.lookupNodeInfo(nodeName)

		log.Printf("providerID for node %s=%s", nodeName, providerid)

		if annotationValue == providerid {
			log.Print("pod's annotation is already set to the right value")
			c.workqueue.Forget(obj)
			return nil
		}

		annotations := p.ObjectMeta.GetAnnotations()
		annotations[c.annotationKey] = providerid
		p.ObjectMeta.SetAnnotations(annotations)

		if _, err := c.clientset.CoreV1().Pods(namespace).Update(context.TODO(), p, metav1.UpdateOptions{}); err != nil {
			c.workqueue.AddRateLimited(obj)
			return fmt.Errorf("could not update pod %s in %s namespace: %v", p.ObjectMeta.Name, namespace, err)
		}

		log.Printf("successfully updated pod %s in namespace %s", p.ObjectMeta.Name, namespace)
		c.workqueue.Forget(obj)

		return nil
	}(obj)

	if err != nil {
		log.Printf("error received processing work item: %v", err)
		return true
	}

	return true
}

func (c Controller) lookupNodeInfo(name string) string {
	info, ok := c.nodeCache[name]
	if !ok {
		log.Printf("node cache miss - could not get info for node %s", name)
		return ""
	}
	return info
}
