# aks-http-proxy-append
A mutation webook to allow you to specify a default no_proxy list for pods and append any values the pod defines.

* Comment out the secret volume stuff in `kubernetes/1-deployment.yaml`
* Run `kubectl apply -f kubernetes/1-deployment.yaml`
* Run `kubectl apply -f kubernetes/2-service.yaml`
* Run `ssl/certgen.sh`
* Uncomment out the secret volume stuff
* Rerun `kubectl apply -f kubernetes/1-deployment.yaml`
* Run `${kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.certificate-authority-data}'}`
* Copy `kubernetes/3-webhook.sample.yaml` to `kubernetes/3-webhook.yaml` and update the cert value
* Run `kubectl apply -f kubernetes/3-webhook.yaml`
* Try creating a new pod and see what the no_proxy list is!

## Resources
* https://github.com/trstringer/kubernetes-mutating-webhook
* https://github.com/alex-leonhardt/k8s-mutate-webhook
* https://github.com/cryptk/kubernetes-mimic/blob/main/deploy/scripts/webhook-create-signed-cert.sh
* https://github.com/kubernetes/kubernetes/blob/release-1.21/test/images/agnhost/webhook/main.go
* https://stackoverflow.com/questions/66419906/how-to-create-a-certificatesigningrequest-with-apiversion-certificates-k8s-io