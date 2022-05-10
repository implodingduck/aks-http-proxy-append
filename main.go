// based on:
// https://github.com/trstringer/kubernetes-mutating-webhook/blob/main/cmd/root.go
// https://github.com/alex-leonhardt/k8s-mutate-webhook
// https://github.com/kubernetes/kubernetes/blob/release-1.21/test/images/agnhost/webhook/
package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"strings"

	jsonpatch "gomodules.xyz/jsonpatch/v2"
	admission "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	klog "k8s.io/klog/v2"
)

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello %q", html.EscapeString(r.URL.Path))
}

func arrayContainsValue(array []string, item string) bool {

	for i := 0; i < len(array); i++ {
		if array[i] == item {
			return true
		}
	}

	return false
}

func updateEnvVariable(p *corev1.Pod, containerIndex int, envIndex int, value string) {
	klog.Info(fmt.Sprintf("Setting Container %d EnvVar %d to: %s", containerIndex, envIndex, value))
	p.Spec.Containers[containerIndex].Env[envIndex].Value = value
}

func handleMutate(w http.ResponseWriter, r *http.Request) {
	klog.Info("lets try to mutate")
	// reading in the body content
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	klog.Info(string(body[:]))
	if err != nil {
		klog.Fatal(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
		return
	}

	klog.Info("parsing admissionReview")
	// converting into the admissionReview object
	admissionReview := admission.AdmissionReview{}
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, fmt.Sprintf("unmarshaling request failed with %s", err))
		return
	}
	// logging out the contents of the admission review
	review, err := json.Marshal(admissionReview)
	klog.Info(string(review[:]))

	// get the pod object from the AdmissionReview
	codecs := serializer.NewCodecFactory(runtime.NewScheme())
	deserializer := codecs.UniversalDeserializer()
	rawRequest := admissionReview.Request.Object.Raw
	pod := corev1.Pod{}

	if _, _, err := deserializer.Decode(rawRequest, nil, &pod); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, fmt.Sprintf("error creating pod object: %v", err))
		return
	}
	myNoProxyArray := []string{"bing.com", "github.com", "ubuntu.com", "microsoft.com"}
	// loop over all the container specs
	for i := 0; i < len(pod.Spec.Containers); i++ {
		container := pod.Spec.Containers[i]
		envArray := container.Env
		// loop over the environment variables
		for x := 0; x < len(envArray); x++ {
			envVar := envArray[x]
			if envVar.Name == "NO_PROXY" || envVar.Name == "no_proxy" {
				envjson, _ := json.Marshal(envVar)
				klog.Info(string(envjson[:]))
				noProxyArr := strings.Split(envVar.Value, ",")
				for y := 0; y < len(myNoProxyArray); y++ {
					myNoProxyVal := myNoProxyArray[y]
					if !arrayContainsValue(noProxyArr, myNoProxyVal) {
						noProxyArr = append(noProxyArr, myNoProxyVal)
					}
				}
				newNoProxyVal := strings.Join(noProxyArr, ",")
				if newNoProxyVal != envVar.Value {
					// this is how we set the new value
					updateEnvVariable(&pod, i, x, newNoProxyVal)
				}
			}

		}
	}

	// log out pod stuff
	podJson, err := json.Marshal(pod)
	klog.Info(string(podJson[:]))

	// generate a patch to modify the deployment
	patches, err := jsonpatch.CreatePatch(rawRequest, podJson)

	// creating the admissionResponse
	admissionResponse := &admission.AdmissionResponse{}
	admissionResponse.Allowed = true
	if len(patches) != 0 {
		patchType := admission.PatchTypeJSONPatch
		admissionResponse.PatchType = &patchType
		admissionResponse.Patch, err = json.Marshal(patches)
	}

	// the rest is kind of boiler plate stuff
	var admissionReviewResponse admission.AdmissionReview
	admissionReviewResponse.Response = admissionResponse
	admissionReviewResponse.SetGroupVersionKind(admissionReview.GroupVersionKind())
	admissionReviewResponse.Response.UID = admissionReview.Request.UID

	resp, err := json.Marshal(admissionReviewResponse)
	klog.Info("the response:")
	klog.Info(string(resp[:]))
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)

}

func main() {
	certFile := "/ssl/cert.pem"
	keyFile := "/ssl/cert.key"
	port := 8443
	klog.Info("server started")
	sCert, serr := tls.LoadX509KeyPair(certFile, keyFile)
	if serr != nil {
		klog.Fatal(serr)
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{sCert},
	}

	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/mutate", handleMutate)

	server := &http.Server{
		Addr:      fmt.Sprintf(":%d", port),
		TLSConfig: config,
	}
	err := server.ListenAndServeTLS("", "")
	if err != nil {
		panic(err)
	}
}
