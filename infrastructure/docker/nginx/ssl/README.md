TLS termination for the infrastructure/docker stack.

Mounted read-only at /etc/nginx/ssl. Place a real certificate pair here before
exposing the stack to the internet:

  server.crt
  server.key

Generate a local development pair with:

  openssl req -x509 -newkey rsa:2048 -nodes -days 365 \
    -keyout server.key -out server.crt -subj "/CN=localhost"

Certificates are intentionally not committed to version control.
