# Simple node.js Example

To run in docker,

```
docker run \
  -it \
  --name node \
  --rm \
  -p 8080:8080 \
  -v $(pwd):/usr/src/app \
  -w /usr/src/app \
  -e GREETING='Good Afternoon' \
  node:8 \
  node app.js
```

To deploy on OpenShift,

```
oc new-build \
  --name simplenode \
  -l app=simplenode \
  --binary \
  -i nodejs

oc start-build \
  simplenode \
  --from-dir=. \
  --follow

oc new-app \
  simplenode \
  -e GREETING='Good Afternoon'

oc set probe \
  deployment/simplenode \
  --readiness \
  

oc expose svc/simplenode
```
