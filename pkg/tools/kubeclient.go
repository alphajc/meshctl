package tools

import (
	"log"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func getKubeRestConfig(kubeconfig string) (*rest.Config, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

// GetKubeClient create or get a kubeclient
func GetKubeClient(kubeconfig string) (*kubernetes.Clientset, error) {
	config, err := getKubeRestConfig(kubeconfig)
	if err != nil {
		log.Fatalf("Failed to create k8s rest client: %s", err)
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create k8s clientset: %s", err)
		return nil, err
	}

	return clientset, nil
}
