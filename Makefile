PROJ=demo
IMAGENAME=ghcr.io/kwkoo/pod-watcher
VERSION=0.1

BASE:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

.PHONY: run test deploy clean deploydemo cleandemo curl logs image

run:
	@echo "Running pod-watcher locally..."
	@cd $(BASE)/pod-watcher && NUMWORKERS=2 ANNOTATIONKEY=providerid KUBECONFIG=~/.kube/config go run .

test:
	@cd $(BASE)/pod-watcher && go test -v .

deploy:
	@oc project $(PROJ) || oc new-project $(PROJ)
	@oc create sa -n $(PROJ) pod-watcher
	@oc create clusterrole pod-watcher --verb=watch,list,update --resource=pods
	@oc create clusterrole node-lister --verb=get,watch,list --resource=nodes
	@oc adm policy add-cluster-role-to-user pod-watcher -z pod-watcher -n $(PROJ) --rolebinding-name=pod-watcher-pod-watcher
	@oc adm policy add-cluster-role-to-user node-lister -z pod-watcher -n $(PROJ) --rolebinding-name=pod-watcher-node-lister
	@oc new-build -n $(PROJ) --name=pod-watcher -l app=pod-watcher --binary --docker-image=ghcr.io/kwkoo/go-toolset-7-centos7:latest
	@/bin/echo -n "waiting for golang builder imagestreamtag..."; \
	while [ `oc get -n $(PROJ) istag/go-toolset-7-centos7:latest --no-headers 2>/dev/null | wc -l` -lt 1 ]; do \
	  /bin/echo -n "."; \
	  sleep 5; \
	done; \
	/bin/echo "done"
	@oc start-build -n $(PROJ) pod-watcher --from-dir=$(BASE)/pod-watcher --follow
	@oc new-app -i pod-watcher -n $(PROJ) -e ANNOTATIONKEY=nodeinfo
	@oc set sa -n $(PROJ) deploy/pod-watcher pod-watcher

clean:
	-@oc delete all -l app=pod-watcher -n $(PROJ)
	-@oc delete clusterrolebinding/pod-watcher-pod-watcher
	-@oc delete clusterrolebinding/pod-watcher-node-lister
	-@oc delete clusterrole/pod-watcher
	-@oc delete clusterrole/node-lister
	-@oc delete sa/pod-watcher -n $(PROJ)

deploydemo:
	@oc new-build --name demo -l app=demo --binary -i nodejs -n $(PROJ)
	@oc start-build demo --from-dir=$(BASE)/demo --follow -n $(PROJ)
	@oc apply -n $(PROJ) -f $(BASE)/demo.yaml
	@echo "demo is now available at http://`oc get -n $(PROJ) route/demo -o jsonpath='{.spec.host}'`"

cleandemo:
	-@oc delete all -l app=demo -n $(PROJ)
	-@IMAGE="`oc get images | grep $(PROJ)/demo | awk '{ print $$1 }'`"; \
	if [ -n "$$IMAGE" ]; then \
	  oc delete image $$IMAGE; \
	fi

curl:
	@curl "http://`oc get -n $(PROJ) route/demo -o jsonpath='{.spec.host}'`"

logs:
	@oc logs -n $(PROJ) -f deploy/pod-watcher

image:
	@docker build --rm -t $(IMAGENAME):$(VERSION) $(BASE)
	@docker push $(IMAGENAME):$(VERSION)