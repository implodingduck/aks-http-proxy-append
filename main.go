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

	admission "k8s.io/api/admission/v1"
	klog "k8s.io/klog/v2"
)

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello %q", html.EscapeString(r.URL.Path))
}

func handleMutate(w http.ResponseWriter, r *http.Request) {
	klog.Info("lets try to mutate")
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
	admissionReview := admission.AdmissionReview{}
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		fmt.Fprintf(w, fmt.Sprintf("unmarshaling request failed with %s", err))
		return
	}
	review, err := json.Marshal(admissionReview)
	klog.Info(string(review[:]))
	admissionResponse := &admission.AdmissionResponse{}
	admissionResponse.Allowed = true

	var admissionReviewResponse admission.AdmissionReview
	admissionReviewResponse.Response = admissionResponse
	admissionReviewResponse.SetGroupVersionKind(admissionReview.GroupVersionKind())
	admissionReviewResponse.Response.UID = admissionReview.Request.UID

	resp, err := json.Marshal(admissionReviewResponse)
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
