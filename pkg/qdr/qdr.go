package qdr

import (
	"fmt"
	"log"
	"regexp"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/skupperproject/skupper/api/types"
	"github.com/skupperproject/skupper/pkg/kube"
	"github.com/skupperproject/skupper/pkg/utils/configs"
)

func AddConnector(connector *types.Connector, qdrDep *appsv1.Deployment) {
	config := kube.FindEnvVar(qdrDep.Spec.Template.Spec.Containers[0].Env, types.TransportEnvConfig)
	if config == nil {
		fmt.Println("Could not retrieve qdr config")
	}
	updated := config.Value + configs.ConnectorConfig(connector)
	kube.SetEnvVarForDeployment(qdrDep, types.TransportEnvConfig, updated)
	kube.AppendSecretVolume(&qdrDep.Spec.Template.Spec.Volumes, &qdrDep.Spec.Template.Spec.Containers[0].VolumeMounts, connector.Name, "/etc/qpid-dispatch-certs/"+connector.Name+"/")
}

func IsInterior(qdr *appsv1.Deployment) bool {
	config := kube.FindEnvVar(qdr.Spec.Template.Spec.Containers[0].Env, types.TransportEnvConfig)
	//match 'mode: interior' in that config
	if config == nil {
		log.Fatal("Could not retrieve qdr config")
	}
	match, _ := regexp.MatchString("mode:[ ]+interior", config.Value)
	return match
}

func GetTransportMode(dep *appsv1.Deployment) types.TransportMode {
	if IsInterior(dep) {
		return types.TransportModeInterior
	} else {
		return types.TransportModeEdge
	}
}

func ListRouterConnectors(mode types.TransportMode, namespace string, cli kubernetes.Interface) []types.Connector {
	var connectors []types.Connector
	secrets, err := cli.CoreV1().Secrets(namespace).List(metav1.ListOptions{LabelSelector: "skupper.io/type=connection-token"})
	if err == nil {
		var role types.ConnectorRole
		var hostKey string
		var portKey string
		if mode == types.TransportModeEdge {
			role = types.ConnectorRoleEdge
			hostKey = "edge-host"
			portKey = "edge-port"
		} else {
			role = types.ConnectorRoleInterRouter
			hostKey = "inter-router-host"
			portKey = "inter-router-port"
		}
		for _, s := range secrets.Items {
			connectors = append(connectors, types.Connector{
				Name: s.ObjectMeta.Name,
				Host: s.ObjectMeta.Annotations[hostKey],
				Port: s.ObjectMeta.Annotations[portKey],
				Role: string(role),
			})
		}
	} else {
		fmt.Println("Could not retrieve connection-token secrets: ", err.Error())
	}
	return connectors
}
