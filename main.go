// based on:
// https://github.com/trstringer/kubernetes-mutating-webhook/blob/main/cmd/root.go
// https://github.com/alex-leonhardt/k8s-mutate-webhook
package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	admission "k8s.io/api/admission/v1"
)

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello %q", html.EscapeString(r.URL.Path))
}

func handleMutate(w http.ResponseWriter, r *http.Request) {

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
	}
	admissionReview := admission.AdmissionReview{}
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		fmt.Fprintf(w, fmt.Sprintf("unmarshaling request failed with %s", err))
	}
	review, err := json.Marshal(admissionReview)
	fmt.Fprintf(w, string(review[:]))
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

	mux := http.NewServeMux()

	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/mutate", handleMutate)

	s := &http.Server{
		Addr:           ":8443",
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1048576
	}

	log.Fatal(s.ListenAndServeTLS("/ssl/cert.pem", "/ssl/cert.key"))

}
