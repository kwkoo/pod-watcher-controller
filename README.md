# Pod Watcher

A controller that watches for pods being scheduled (i.e. the `PodScheduled` condition type).

If the pod contains a certain annotation, the controller will

* Lookup the node that the pod is scheduled on (`.spec.nodeName`)
* Lookup the node's `.spec.providerID`
* Annotate the pod with the node's `providerID`

The pod can utilize this information by mounting the annotation as a volume using the downward API. Refer to `demo.yaml` for more info on how this is done.

Sample `providerID`:

```
aws:///ap-southeast-1a/i-06fbbd699deb4ebc4
```

Note: The node `providerID`s are cached when the controller starts up. If nodes are added after the controller has started, the controller will not know about the `providerID`s of those nodes.

## Installation

1. Login to OpenShift using `oc login`

1. Build and install the `pod-watcher`:

	```
	make deploy
	```

1. Deploy the demo app:

	```
	make deploydemo
	```

1. After the demo has been deployed, access the demo app with:

	```
	curl http://$(oc get -n demo route/demo -o jsonpath='{.spec.host}')
	```

The demo app should print out the `providerID` of the node that it is deployed on.


## Resources

* [How to write Kubernetes custom controllers in Go](https://medium.com/speechmatics/how-to-write-kubernetes-custom-controllers-in-go-8014c4a04235)
* [Kubernetes sample-controller](https://github.com/kubernetes/sample-controller)
* [Building stuff with the Kubernetes API](https://medium.com/programming-kubernetes/building-stuff-with-the-kubernetes-api-part-4-using-go-b1d0e3c1c899)
* [Kubewatch, an example of Kubernetes custom controller](https://engineering.bitnami.com/articles/kubewatch-an-example-of-kubernetes-custom-controller.html)
* [A deep dive into Kubernetes controllers](https://engineering.bitnami.com/articles/a-deep-dive-into-kubernetes-controllers.html)
* [Extend Kubernetes via a shared informer](https://www.cncf.io/blog/2019/10/15/extend-kubernetes-via-a-shared-informer/)
