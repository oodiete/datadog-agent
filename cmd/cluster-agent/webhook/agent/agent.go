// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

// Package agent implements the api endpoints for the `/agent` prefix.
// This group of endpoints is meant to provide high-level functionalities
// at the agent level.
package agent

import (
	"encoding/json"
	"fmt"
	"github.com/DataDog/datadog-agent/cmd/agent/common/jsonpatch"
	v1 "github.com/DataDog/datadog-agent/cmd/cluster-agent/api/v1"
	"github.com/DataDog/datadog-agent/pkg/clusteragent"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/gorilla/mux"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

// SetupHandlers adds the specific handlers for cluster agent endpoints
func SetupHandlers(r *mux.Router, sc clusteragent.ServerContext) {
	r.HandleFunc("/application-mutating-webhook", getApplicationMutatingWebhook).Methods("POST")

	// Install versioned apis
	v1.Install(r.PathPrefix("/api/v1").Subrouter(), sc)
}

func getApplicationMutatingWebhook(w http.ResponseWriter, r *http.Request) {
	log.Error("got a webhook request!") // TODO remove me

	decoder := json.NewDecoder(r.Body)

	// The AdmissionReview that was sent to the webhook
	requestedAdmissionReview := admissionv1beta1.AdmissionReview{}
	err := decoder.Decode(&requestedAdmissionReview)
	if err != nil {
		log.Errorf("Unable to parse request: %s", err)
		body, _ := json.Marshal(map[string]string{"error": err.Error()})
		http.Error(w, string(body), 500)
		return
	}

	// The AdmissionReview that will be returned
	responseAdmissionReview := admissionv1beta1.AdmissionReview{}

	// pass to admitFunc
	responseAdmissionReview.Response = mutatePods(requestedAdmissionReview)

	// Return the same UID
	responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID

	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		log.Error(err)
	}
	if _, err := w.Write(respBytes); err != nil {
		log.Error(err)
	}
}

// toAdmissionResponse is a helper function to create an AdmissionResponse
// with an embedded error
func toAdmissionResponse(err error) *admissionv1beta1.AdmissionResponse {
	return &admissionv1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}

func mutatePods(ar admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	log.Debug("Processing Mutating Webhook")

	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if ar.Request.Resource != podResource {
		log.Errorf("Expect resource to be %s, got %v", podResource, ar.Request.Resource)

		return &admissionv1beta1.AdmissionResponse{Allowed: true}
	}

	pod := corev1.Pod{}
	if err := json.Unmarshal(ar.Request.Object.Raw, &pod); err != nil {
		log.Errorf("Could not retrieve Pod from request: %v", err)
		return toAdmissionResponse(err)
	}

	patch := mutatePod(pod)

	reviewResponse := mutateResponse(patch)

	log.Infof("Webhook ReviewResponse: %v", reviewResponse)

	return reviewResponse
}

func mutateResponse(patch jsonpatch.Patch) *admissionv1beta1.AdmissionResponse {
	if patch == nil {
		return &admissionv1beta1.AdmissionResponse{Allowed: true}
	}

	log.Infof("Mutating JSON Patch: %v", patch)

	bs, _ := json.Marshal(patch)
	patchType := admissionv1beta1.PatchTypeJSONPatch
	return &admissionv1beta1.AdmissionResponse{
		Allowed:   true,
		Patch:     bs,
		PatchType: &patchType,
	}
}

// NewEnvMutator creates a new mutator which adds environment
// variables to pods
func getEnvMutator() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: "DD_AGENT_HOST",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.hostIP",
				},
			},
		},
		{
			Name:  "DEV_VERSION",
			Value: "dev-1",
		},
	}
}

func mutatePod(pod corev1.Pod) (jsonpatch.Patch) {
	var envVariables = getEnvMutator()

	containerLists := []struct {
		field      string
		containers []corev1.Container
	}{
		{"initContainers", pod.Spec.InitContainers},
		{"containers", pod.Spec.Containers},
	}

	var patch jsonpatch.Patch

	for _, s := range containerLists {
		field, containers := s.field, s.containers
		for i, container := range containers {
			if len(container.Env) == 0 {
				patch = append(patch, jsonpatch.Add(
					fmt.Sprint("/spec/", field, "/", i, "/env"),
					[]interface{}{},
				))
			}

			remainingEnv := make([]corev1.EnvVar, len(container.Env))
			copy(remainingEnv, container.Env)

		injectedEnvLoop:
			for envPos, def := range envVariables {
				for pos, v := range remainingEnv {
					if v.Name == def.Name {
						if currPos, destPos := envPos+pos, envPos; currPos != destPos {
							// This should ideally be a `move` operation but due to a bug in the json-patch's
							// implementation of `move` operation, we explicitly use `remove` followed by `add`.
							// see, https://github.com/evanphx/json-patch/pull/73
							// This is resolved in json-patch `v4.2.0`, which is pulled by Kubernetes `1.14.3` clusters.
							// https://github.com/kubernetes/kubernetes/blob/v1.14.3/Godeps/Godeps.json#L1707-L1709
							// TODO: Use a `move` operation, once all clusters are on `1.14.3+`
							patch = append(patch,
								jsonpatch.Remove(
									fmt.Sprint("/spec/", field, "/", i, "/env/", currPos),
								),
								jsonpatch.Add(
									fmt.Sprint("/spec/", field, "/", i, "/env/", destPos),
									v,
								))
						}
						remainingEnv = append(remainingEnv[:pos], remainingEnv[pos+1:]...)
						continue injectedEnvLoop
					}
				}

				patch = append(patch, jsonpatch.Add(
					fmt.Sprint("/spec/", field, "/", i, "/env/", envPos),
					def,
				))
			}
		}
	}
	return patch
}
