#!/bin/bash

# set -e

# Updated version of https://github.com/cryptk/kubernetes-mimic/blob/main/deploy/scripts/webhook-create-signed-cert.sh
# Accounting for https://stackoverflow.com/a/68665226

service=akshttpproxyappend
secret=akshttpproxyappend-certs
namespace=default

kubectl --namespace ${namespace} get secret ${secret}
if [ $? -eq 0 ]; then
  echo "Secret ${secret} already exists in namespace ${namespace}, exiting"
  exit 0
fi

if [ ! -x "$(command -v openssl)" ]; then
    echo "openssl not found"
    exit 1
fi

csrName=${service}.${namespace}.svc.cluster.local
tmpdir=$(mktemp -d)
echo "creating certs in tmpdir ${tmpdir} "

cat <<EOF >> "${tmpdir}"/csr.conf
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
DNS.4 = ${service}.${namespace}.svc.cluster.local
EOF

openssl genrsa -out "${tmpdir}"/server-key.pem 2048
openssl req -new -key "${tmpdir}"/server-key.pem -subj "/CN=system:node:${service}.${namespace}.svc/OU=system:nodes/O=system:nodes" -out "${tmpdir}"/server.csr -config "${tmpdir}"/csr.conf

echo "cleaning up old csr ${csrName}"
# clean-up any previously created CSR for our service. Ignore errors if not present.
kubectl delete csr ${csrName} 2>/dev/null || true

echo "creating server cert/key CSR and add it to k8s"
# create  server cert/key CSR and  send to k8s API
cat <<EOF | kubectl create -f -
apiVersion: certificates.k8s.io/v1
kind: CertificateSigningRequest
metadata:
  name: ${csrName}
spec:
  groups:
  - system:authenticated
  request: $(< "${tmpdir}"/server.csr base64 | tr -d '\n')
  signerName: kubernetes.io/kubelet-serving
  usages:
  - digital signature
  - key encipherment
  - server auth
EOF

# verify CSR has been created
while true; do
    if kubectl get csr ${csrName}; then
        break
    else
        sleep 1
    fi
done

# approve and fetch the signed certificate
kubectl certificate approve ${csrName}
# verify certificate has been signed
for _ in $(seq 20); do
    serverCert=$(kubectl get csr ${csrName} -o jsonpath='{.status.certificate}')
    if [[ ${serverCert} != '' ]]; then
        break
    fi
    sleep 1
done
if [[ ${serverCert} == '' ]]; then
    echo "ERROR: After approving csr ${csrName}, the signed certificate did not appear on the resource. Giving up after 10 attempts." >&2
    exit 1
fi
echo "${serverCert}" | openssl base64 -d -A -out "${tmpdir}"/server-cert.pem


# create the secret with CA cert and server cert/key
kubectl create secret generic ${secret} \
        --from-file=cert.key="${tmpdir}"/server-key.pem \
        --from-file=cert.pem="${tmpdir}"/server-cert.pem \
        --dry-run=client -o yaml |
    kubectl -n ${namespace} apply -f -
