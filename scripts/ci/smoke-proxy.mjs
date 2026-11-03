// Local test fixture matching Facet's backend/frontend routing boundary.
// Release images use docker/Caddyfile; this avoids a host-level Caddy install.
import http from 'node:http';

const host = process.env.SMOKE_HOST || '127.0.0.1';
const proxyPort = Number(process.env.SMOKE_PROXY_PORT || 18080);
const backendPort = Number(process.env.SMOKE_BACKEND_PORT || 18090);
const frontendPort = Number(process.env.SMOKE_FRONTEND_PORT || 13000);
const backend = ['/api/', '/_/', '/rss.xml', '/talks.ics'];

http.createServer((request, response) => {
  const path = request.url || '/';
  const isBackend = backend.some((prefix) => path.startsWith(prefix));
  const headers = {
    ...request.headers,
    'x-forwarded-proto': 'http',
    'x-forwarded-host': request.headers.host,
    'x-forwarded-for': request.socket.remoteAddress,
  };
  delete headers['x-internal'];
  const upstream = http.request({
    hostname: host,
    port: isBackend ? backendPort : frontendPort,
    path,
    method: request.method,
    headers,
  }, (result) => {
    response.writeHead(result.statusCode, result.headers);
    result.pipe(response);
  });
  upstream.on('error', () => {
    response.writeHead(502);
    response.end('Smoke-test upstream unavailable');
  });
  request.pipe(upstream);
}).listen(proxyPort, host);
