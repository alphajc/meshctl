package tools

import (
    "log"

    versionedclient "istio.io/client-go/pkg/clientset/versioned"
)

// GetIstioClient to get a istio clientset
func GetIstioClient(kubeconfig string) (*versionedclient.Clientset, error) {

    config, err := getKubeRestConfig(kubeconfig)
    ic, err := versionedclient.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create istio client: %s", err)
        return nil, err
	}

    return ic, nil
}
