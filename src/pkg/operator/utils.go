package operator

import "k8s.io/client-go/rest"

func isInCluster() bool {
	_, err := rest.InClusterConfig()
	return err == nil
}
