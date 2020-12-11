package app

import (
	"flag"
	"fmt"

	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
)

type ServiceVersion struct {
    DestinationRule networkingv1beta1.DestinationRule
}

func Test() {
    config := flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
    flag.Parse()
    fmt.Println(*config)
}
