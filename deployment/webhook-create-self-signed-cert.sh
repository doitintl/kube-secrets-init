#!/bin/bash

set -e

usage() {
    cat <<EOF
Generate certificate suitable for use with an secrets-init webhook service.

This script generates self-signed certificate for the webhook. See
https://www.velotio.com/engineering-blog/managing-tls-certificate-for-kubernetes-admission-webhook
detailed explantion and additional instructions.

The server key/cert k8s CA cert are stored in a k8s secret.

usage: ${0} [OPTIONS]

The following flags are required.

       --service          Service name of webhook.
       --namespace        Namespace where webhook service and secret reside.
       --secret           Secret name for CA certificate and server certificate/key pair.
EOF
    exit 1
}

while [[ $# -gt 0 ]]; do
    case ${1} in
        --service)
            service="$2"
            shift
            ;;
        --secret)
            secret="$2"
            shift
            ;;
        --namespace)
            namespace="$2"
            shift
            ;;
        *)
            usage
            ;;
    esac
    shift
done

[ -z ${service} ] && service=secrets-init-webhook-svc
[ -z ${secret} ] && secret=secrets-init-webhook-certs
[ -z ${namespace} ] && namespace=default

if [ ! -x "$(command -v openssl)" ]; then
    echo "openssl not found"
    exit 1
fi

csrName=${service}.${namespace}
tmpdir=$(mktemp -d)
echo "creating certs in tmpdir ${tmpdir} "

cat <<EOF >> ${tmpdir}/csr.conf
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = ${service}
DNS.2 = ${service}.${namespace}
DNS.3 = ${service}.${namespace}.svc
EOF

# create CA and Server key/certificate
openssl genrsa -out ${tmpdir}/ca.key 2048
openssl req -x509 -newkey rsa:2048 -key ${tmpdir}/ca.key -out ${tmpdir}/ca.crt -days 1825 -nodes -subj "/CN=${service}.${namespace}.svc"

# create server key/certificate
openssl genrsa -out ${tmpdir}/server.key 2048
openssl req -new -key ${tmpdir}/server.key -subj "/CN=${service}.${namespace}.svc" -out ${tmpdir}/server.csr -config ${tmpdir}/csr.conf

# Self sign
openssl x509 -extensions v3_req -req -days 1825 -in ${tmpdir}/server.csr -CA ${tmpdir}/ca.crt -CAkey ${tmpdir}/ca.key -CAcreateserial -out ${tmpdir}/server.crt -extfile ${tmpdir}/csr.conf

# create the secret with CA cert and server cert/key
kubectl create secret generic ${secret} \
        --from-file=key.pem=${tmpdir}/server.key \
        --from-file=cert.pem=${tmpdir}/server.crt \
        --dry-run=client -o yaml |
    kubectl -n ${namespace} apply -f -

# -a means base64 encode
caBundle=$(cat ${tmpdir}/ca.crt | openssl enc -a -A)

echo "Encoded CA:"
echo -e "${caBundle} \n"