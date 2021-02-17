openssl req -newkey rsa:2048 \
    -nodes \
    -keyout ./certs/server.key \
    -x509 -days 365 \
    -out ./certs/server.crt \
    -subj "/C=US/ST=New York/L=Brooklyn/O=Example Brooklyn Company/CN=examplebrooklyn.com" \
    -addext "subjectAltName=DNS:127.0.0.1,IP:127.0.0.1"


