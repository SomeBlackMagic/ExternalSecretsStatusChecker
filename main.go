package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	// Retrieve the namespace and resource name from command-line arguments
	namespace := flag.String("namespace", "", "Namespace of the ExternalSecret")
	name := flag.String("name", "", "Name of the ExternalSecret")
	flag.Parse()

	if *namespace == "" || *name == "" {
		fmt.Println("Usage: ./program -namespace=<namespace> -name=<name>")
		os.Exit(1)
	}

	// Read KUBECONFIG from environment variables or from the home directory
	kubeconfigEnv := os.Getenv("KUBECONFIG")
	var kubeconfig string
	if kubeconfigEnv != "" {
		kubeconfig = kubeconfigEnv
	} else if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	var config *rest.Config
	var err error

	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		fmt.Printf("Error building kubeconfig: %v\n", err)
		os.Exit(1)
	}

	// Create client sets
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Error creating Kubernetes clientset: %v\n", err)
		os.Exit(1)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Printf("Error creating dynamic client: %v\n", err)
		os.Exit(1)
	}

	// Start watching events in a separate goroutine
	go watchEvents(clientset, *namespace, *name)

	// Check the status of the ExternalSecret
	checkStatus(dynamicClient, *namespace, *name)
}

func watchEvents(clientset *kubernetes.Clientset, namespace, name string) {
	fmt.Printf("Watching events for ExternalSecret %s in namespace %s...\n", name, namespace)
	fieldSelector := fields.AndSelectors(
		fields.OneTermEqualSelector("involvedObject.kind", "ExternalSecret"),
		fields.OneTermEqualSelector("involvedObject.name", name),
	).String()

	listOptions := metav1.ListOptions{
		FieldSelector: fieldSelector,
	}

	for {
		watcher, err := clientset.CoreV1().Events(namespace).Watch(context.TODO(), listOptions)
		if err != nil {
			fmt.Printf("Error watching events: %v\n", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for event := range watcher.ResultChan() {
			if e, ok := event.Object.(*corev1.Event); ok {
				fmt.Printf("Event: %s - %s: %s\n", e.LastTimestamp, e.Reason, e.Message)
			}
		}
	}
}

func checkStatus(dynamicClient dynamic.Interface, namespace, name string) {
	// Define the GroupVersionResource for ExternalSecret
	externalSecretGVR := schema.GroupVersionResource{
		Group:    "external-secrets.io",
		Version:  "v1beta1",
		Resource: "externalsecrets",
	}

	for {
		// Get the ExternalSecret resource
		unstructuredES, err := dynamicClient.Resource(externalSecretGVR).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Error getting ExternalSecret: %v\n", err)
		} else {
			if isReady(unstructuredES) {
				fmt.Printf("ExternalSecret %s has reached Ready state.\n", name)
				os.Exit(0)
			} else {
				fmt.Printf("Waiting... Current status conditions: %v\n", getConditions(unstructuredES))
			}
		}
		time.Sleep(5 * time.Second)
	}
}

func isReady(unstructuredES *unstructured.Unstructured) bool {
	conditions := getConditions(unstructuredES)
	for _, condition := range conditions {
		if condition.Type == "Ready" && condition.Status == "True" {
			return true
		}
	}
	return false
}

type Condition struct {
	Type               string `json:"type"`
	Status             string `json:"status"`
	LastTransitionTime string `json:"lastTransitionTime"`
	Reason             string `json:"reason"`
	Message            string `json:"message"`
}

func getConditions(unstructuredES *unstructured.Unstructured) []Condition {
	status, found, err := unstructured.NestedMap(unstructuredES.Object, "status")
	if !found || err != nil {
		return []Condition{}
	}

	conditionsInterface, found, err := unstructured.NestedSlice(status, "conditions")
	if !found || err != nil {
		return []Condition{}
	}

	var conditions []Condition
	for _, c := range conditionsInterface {
		conditionMap, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		conditionBytes, err := json.Marshal(conditionMap)
		if err != nil {
			continue
		}
		var condition Condition
		err = json.Unmarshal(conditionBytes, &condition)
		if err != nil {
			continue
		}
		conditions = append(conditions, condition)
	}
	return conditions
}
