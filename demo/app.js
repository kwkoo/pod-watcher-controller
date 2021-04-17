const http = require('http');
const fs = require('fs');

const hostname = '0.0.0.0';
const port = 8080;

var filename = '/etc/podinfo/data';

const server = http.createServer((req, res) => {
  res.setHeader('Content-Type', 'text/plain');

  // do not log requests from Kubernetes probes
  //
  if (req.headers['user-agent'] != null && req.headers['user-agent'].startsWith('kube-probe')) {
      res.end('OK');
      return;
  }

  res.statusCode = 200;
  fs.readFile(filename, 'utf8', function (err, data) {
    if (err) {
      res.end(`error: ${err}`);
    } else {
      res.end(`node info: ${data}`)
    }
  });
});

server.listen(port, hostname, () => {
  console.log(`Server running at http://${hostname}:${port}/`);
});