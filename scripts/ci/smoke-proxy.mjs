// Local test fixture matching Facet's backend/frontend routing boundary.
// Release images use docker/Caddyfile; this avoids a host-level Caddy install.
import http from 'node:http';

const backend = ['/api/', '/_/', '/rss.xml', '/talks.ics'];
http.createServer((request, response) => {
  const isBackend = backend.some((prefix) => request.url.startsWith(prefix));
  const upstream = http.request({
    hostname: '127.0.0.1',
    port: isBackend ? 18090 : 13000,
    path: request.url,
    method: request.method,
    headers: request.headers,
  }, (result) => {
    response.writeHead(result.statusCode, result.headers);
    result.pipe(response);
  });
  upstream.on('error', () => {
    response.writeHead(502);
    response.end('Smoke-test upstream unavailable');
  });
  request.pipe(upstream);
}).listen(18080, '127.0.0.1');
