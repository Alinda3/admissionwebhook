package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// A struct that will hold the patch Operations
type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// Root path for our API
func HandleRoot(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("HandlerRoot!"))
}

// The mutation logic
func HandleMutate(w http.ResponseWriter, r *http.Request) {

	// Receive the admissionreview
	body, err := io.ReadAll(r.Body)

	var admissionReviewReq v1beta1.AdmissionReview

	if _, _, err := universalDeserializer.Decode(body, nil, &admissionReviewReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("could not deserialize request: %v\n", err)
	} else if admissionReviewReq.Request == nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("Malformed Admission Review: Request is nil\n")
	}

	// Read the k8s configmap from the AKS and push it's value to the map
	cmNamespace := os.Getenv("cmNamespace")
	cmName := os.Getenv("cmName")
	log.Printf("Reading k8s configMaps %v\n", cmName)
	cm, err := clientSet.CoreV1().ConfigMaps(cmNamespace).Get(context.TODO(), cmName, metav1.GetOptions{})
	if err != nil {
		log.Printf("Could not read configMap %v\n", err)
	}

	log.Printf("Type: %v \t Event: %v \t Name: %v \n",
		admissionReviewReq.Request.Kind,
		admissionReviewReq.Request.Operation,
		admissionReviewReq.Request.Name,
	)

	rawCmData := cm.Data

	// Initialize the result map
	cmData := make(map[string]map[string]string)

	// Convert the ConfigMap to the desired map structure
	for key, value := range rawCmData {
		cmData[key] = make(map[string]string)
		cmData[key] = parseConfigMapData(value)
	}

	// this will hold our new deployment object that we will send back to the API server
	var deployment appsv1.Deployment

	err = json.Unmarshal(admissionReviewReq.Request.Object.Raw, &deployment)

	if err != nil {
		log.Printf("Could not unmarshal pod on admission request: %v\n", err)
	}

	// check if we already have nodeselector so we dont overwrite them by mistake
	existingNodeSelector := deployment.Spec.Template.Spec.NodeSelector

	// using the configmap find out what new nodeselector we should inject if any
	newNodeSelector := make(map[string]string)
	labelToCheck := deployment.ObjectMeta.Labels[os.Getenv("labelToCheck")]
	value, exists := cmData[labelToCheck]
	if exists {
		for k, v := range value {
			newNodeSelector[k] = v
		}

		if len(existingNodeSelector) != 0 {
			// merge old and new nodeselectors
			for k, v := range existingNodeSelector {
				newNodeSelector[k] = v
			}
		}
	} else {
		log.Printf("%v does not match any of the configMap %v keys!\n", labelToCheck, cmName)
	}

	// initialize our patch
	var patches []patchOperation
	patches = append(patches, patchOperation{
		Op:    "replace",
		Path:  "/spec/template/spec/nodeSelector",
		Value: newNodeSelector,
	})

	patchBytes, err := json.Marshal(patches)

	if err != nil {
		log.Printf("Could not marshal JSON patch: %v\n", err)
	}

	// create a admisionresponse and validate it at the same time
	// future enhancement can be to split the validation process and deploy it as a ValidationWebhook by it's own
	// for example we can double check that indeed the nodeselector matches what exists in the AKS  ;)
	admissionReviewResponse := v1beta1.AdmissionReview{
		Response: &v1beta1.AdmissionResponse{
			UID:     admissionReviewReq.Request.UID,
			Allowed: true,
		},
	}

	admissionReviewResponse.Response.Patch = patchBytes

	bytes, err := json.Marshal(&admissionReviewResponse)
	if err != nil {
		log.Printf("Marshaling response: %v\n", err)
	}

	w.Write(bytes)
}
