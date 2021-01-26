package app

import (
	"context"
	"log"
    "fmt"
    "errors"
	"strings"

	"github.com/alphajc/meshctl/pkg/tools"
	"istio.io/client-go/pkg/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
    "istio.io/api/networking/v1beta1"
)

type ListArguments struct {
    Namespace string
    Kubeconfig string
}

type Clientset struct {
    kc *kubernetes.Clientset
    ic *versioned.Clientset
}

type AppVersion struct {
    Namespace string
    Service string
    Version string
}

// CreateClientset to get kubernetes clientset and istio clientset
func CreateClientset(kubeconfig string) (clienset *Clientset, err error) {
    kc, err := tools.GetKubeClient(kubeconfig)
	if err != nil {
		return
	}

    ic, err := tools.GetIstioClient(kubeconfig)
	if err != nil {
		return
	}

    clienset = &Clientset{kc, ic}
    return
}

// CheckAppVersion to check if the application and its version exist
func (cs *Clientset) CheckAppVersion(appVersion *AppVersion) (err error) {
    subsetName := strings.Replace(appVersion.Version, ".", "-", -1)

    // get service
    _, err = cs.kc.CoreV1().Services(appVersion.Namespace).Get(context.TODO(), appVersion.Service, metav1.GetOptions{})
	if err != nil {
		return
	}

    // get deployment
    deploymentName := strings.Join([]string{
        appVersion.Service,
        subsetName,
    }, "-")
    _, err = cs.kc.AppsV1().Deployments(appVersion.Namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		return
	}

    // get destinationrule
    dr, err := cs.ic.NetworkingV1beta1().DestinationRules(appVersion.Namespace).Get(context.TODO(), appVersion.Service, metav1.GetOptions{})
	if err != nil {
		return
	}
    for _, subset := range dr.Spec.GetSubsets() {
        if subset.Labels["version"] == appVersion.Version {
            err = errors.New(
                fmt.Sprintf(
                    "Subset already exist (service:%s subset:%s version:%s)",
                    appVersion.Service,
                    subset.Name,
                    subset.Labels["version"],
                ),
            )
            return
        }
    }
    //get virtualservice
    vs, err := cs.ic.NetworkingV1beta1().VirtualServices(appVersion.Namespace).Get(context.TODO(), appVersion.Service, metav1.GetOptions{})
	if err != nil {
		return
	}
    for _, httpRoute := range vs.Spec.GetHttp() {
        routeRecord := "httpRoute:"
        for i, m := range httpRoute.GetMatch() {
            routeRecord = fmt.Sprintf("%s\nmatch[%d]:", routeRecord, i)
            if m.Authority != nil {
                routeRecord = fmt.Sprintf("%s\n\tpath:%s", routeRecord, m.Authority)
            }
            routeRecord = fmt.Sprintf("%s\n\tGateway: ", routeRecord)
            for _, gateway := range m.Gateways {
                routeRecord = fmt.Sprintf("%s%s ", routeRecord, gateway)
            }
            routeRecord = fmt.Sprintf("%s\n\tHeaders: ", routeRecord)
            for k, v := range m.Headers {
                routeRecord = fmt.Sprintf("%s%s->%s ", routeRecord, k, v)
            }
        }
        for j, r := range httpRoute.GetRoute() {
            routeRecord = fmt.Sprintf("%s\nRoute[%d] (host=%s subset=%s)", routeRecord, j, r.Destination.GetHost(), r.Destination.GetSubset())
        }
        log.Println(routeRecord)
    }

    return
}

// AddAppVersion to add a new version
func (cs *Clientset) AddAppVersion(appVersion *AppVersion) (err error) {
    if err = cs.CheckAppVersion(appVersion); err != nil {
        return
    }
    subsetName := strings.Replace(appVersion.Version, ".", "-", -1)
    dr, err := cs.ic.NetworkingV1beta1().DestinationRules(appVersion.Namespace).Get(context.TODO(), appVersion.Service, metav1.GetOptions{})
    dr.Spec.Subsets = append(
        dr.Spec.Subsets,
        &v1beta1.Subset{
            Name: subsetName,
            Labels: map[string]string{"version": appVersion.Version},
        },
    )

    dr, err = cs.ic.NetworkingV1beta1().DestinationRules(appVersion.Namespace).Update(context.TODO(), dr, metav1.UpdateOptions{})
    log.Println(dr.Spec.Subsets)

    vs, err := cs.ic.NetworkingV1beta1().VirtualServices(appVersion.Namespace).Get(context.TODO(), appVersion.Service, metav1.GetOptions{})
    vs.Spec.Http = append(
        []*v1beta1.HTTPRoute{
            {
                Match: []*v1beta1.HTTPMatchRequest{
                    {
                        Headers: map[string]*v1beta1.StringMatch{
                            "x-weike-forward": {
                                MatchType: &v1beta1.StringMatch_Exact{
                                    Exact: appVersion.Version,
                                },
                            },
                        },
                    },
                },
                Route: []*v1beta1.HTTPRouteDestination{
                    {
                        Destination: &v1beta1.Destination{
                            Host: fmt.Sprintf("%s.%s.svc.cluster.local", appVersion.Service, appVersion.Namespace),
                            Subset: subsetName,
                        },
                    },
                },
            },
        },
        vs.Spec.Http...,
    )
    vs, err = cs.ic.NetworkingV1beta1().VirtualServices(appVersion.Namespace).Update(context.TODO(), vs, metav1.UpdateOptions{})
    log.Println(vs.Spec.Http)
    return
}

// RemoveVersion to remove a abandoned version
func (cs *Clientset) RemoveAppVersion(appVersion *AppVersion) (err error) {
    subsetName := strings.Replace(appVersion.Version, ".", "-", -1)

    vs, err := cs.ic.NetworkingV1beta1().VirtualServices(appVersion.Namespace).Get(context.TODO(), appVersion.Service, metav1.GetOptions{})
    httpRoutes := make([]*v1beta1.HTTPRoute, 0, 5)
    for i, httpRoute := range vs.Spec.Http {
        isCurrentVersion := false
        for _, route := range httpRoute.Route {
            if route.Destination.Subset == subsetName {
                isCurrentVersion = true
                break
            }
        }
        if isCurrentVersion {
            httpRoutes = append(httpRoutes, vs.Spec.Http[i+1:]...)
            break
        }
        httpRoutes = append(httpRoutes, httpRoute)
    }
    log.Println(httpRoutes)
    vs.Spec.Http = httpRoutes
    vs, err = cs.ic.NetworkingV1beta1().VirtualServices(appVersion.Namespace).Update(context.TODO(), vs, metav1.UpdateOptions{})

    dr, err := cs.ic.NetworkingV1beta1().DestinationRules(appVersion.Namespace).Get(context.TODO(), appVersion.Service, metav1.GetOptions{})
    subsets := make([]*v1beta1.Subset, 0, 5)
    for i, subset := range dr.Spec.Subsets {
        if subset.Name == subsetName {
            subsets = append(subsets, dr.Spec.Subsets[i+1:]...)
            break
        }
        subsets = append(subsets, subset)
    }
    log.Println(subsets)
    dr.Spec.Subsets = subsets
    dr, err = cs.ic.NetworkingV1beta1().DestinationRules(appVersion.Namespace).Update(context.TODO(), dr, metav1.UpdateOptions{})

    return
}
