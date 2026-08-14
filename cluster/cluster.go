// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ComputeClass int8

const (
	ComputeClassGeneralPurpose ComputeClass = 0
	ComputeClassBalanced       ComputeClass = 1
	ComputeClassScaleout       ComputeClass = 2
	ComputeClassScaleoutArm    ComputeClass = 3
	ComputeClassPerformance    ComputeClass = 4
	ComputeClassAccelerator    ComputeClass = 5
	ComputeClassGPUPod         ComputeClass = 6
)

var ComputeClasses = [7]string{
	"General-purpose",
	"Balanced",
	"Scale-out",
	"Scale-out arm64",
	"Performance",
	"Accelerator",
	"GPU Pod",
}

func (c ComputeClass) String() string {
	if int(c) >= 0 && int(c) < len(ComputeClasses) {
		return ComputeClasses[c]
	}
	return "Unknown"
}

func (c ComputeClass) Badge() string {
	switch c {
	case ComputeClassGeneralPurpose:
		return "General-purpose"
	case ComputeClassBalanced:
		return "Balanced"
	case ComputeClassScaleout:
		return "Scale-out"
	case ComputeClassScaleoutArm:
		return "Scale-out ARM"
	case ComputeClassPerformance:
		return "Performance"
	case ComputeClassAccelerator:
		return "Accelerator"
	case ComputeClassGPUPod:
		return "GPU Pod"
	default:
		return "Default"
	}
}

type Workload struct {
	Name              string       `json:"name"`
	Namespace         string       `json:"namespace,omitempty"`
	NodeName          string       `json:"node_name"`
	Containers        int          `json:"containers"`
	Cpu               int64        `json:"cpu_mcpu"`
	Memory            int64        `json:"memory_mib"`
	Storage           int64        `json:"storage_mib"`
	AcceleratorType   string       `json:"accelerator_type,omitempty"`
	AcceleratorAmount int64        `json:"accelerator_amount,omitempty"`
	Cost              float64      `json:"cost_hourly"`
	ComputeClass      ComputeClass `json:"compute_class"`
}

type Node struct {
	Name         string     `json:"name"`
	Workloads    []Workload `json:"workloads"`
	InstanceType string     `json:"instance_type"`
	Region       string     `json:"region"`
	Spot         bool       `json:"spot"`
	Cost         float64    `json:"cost_hourly"`
	Accelerator  string     `json:"accelerator,omitempty"`
}

type ClusterInfo struct {
	Name          string `json:"name"`
	Project       string `json:"project"`
	Location      string `json:"location"`
	Status        string `json:"status"`
	MasterVersion string `json:"master_version"`
	NodeCount     int    `json:"node_count"`
	WorkloadCount int    `json:"workload_count"`
}

func GetKubeConfig(customPath string) (*rest.Config, string, error) {
	kubeConfigPath := customPath
	if kubeConfigPath == "" {
		if envPath := os.Getenv("KUBECONFIG"); envPath != "" {
			kubeConfigPath = envPath
		} else {
			userHomeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, "", fmt.Errorf("error getting user home directory: %w", err)
			}
			kubeConfigPath = filepath.Join(userHomeDir, ".kube", "config")
		}
	}

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, "", fmt.Errorf("error loading kubernetes configuration from %s: %w", kubeConfigPath, err)
	}

	return kubeConfig, kubeConfigPath, nil
}

func GetCurrentContext(kubeConfigPath string, overrideContext string) ([]string, error) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: overrideContext,
		}).RawConfig()

	if err != nil {
		return nil, fmt.Errorf("error getting kubernetes current context: %w", err)
	}

	activeContext := overrideContext
	if activeContext == "" {
		activeContext = config.CurrentContext
	}

	if activeContext == "" {
		return nil, fmt.Errorf("no active kubernetes context found in %s", kubeConfigPath)
	}

	// GKE contexts usually follow the format: gke_PROJECT_LOCATION_CLUSTER
	parts := strings.Split(activeContext, "_")
	if len(parts) >= 4 {
		return parts, nil
	}

	// Fallback for non-standard context names
	return []string{"gke", "current-project", "current-location", activeContext}, nil
}

func GetClusterNodes(ctx context.Context, clientset kubernetes.Interface) (map[string]Node, error) {
	nodes := make(map[string]Node)

	clusterNodes, err := ListNodes(ctx, clientset)
	if err != nil {
		return nil, fmt.Errorf("error getting nodes: %w", err)
	}

	for _, clusterNode := range clusterNodes.Items {
		nodes[clusterNode.Name] = Node{
			Name:         clusterNode.Name,
			Region:       clusterNode.Labels["topology.kubernetes.io/region"],
			Spot:         clusterNode.Labels["cloud.google.com/gke-spot"] == "true",
			Accelerator:  clusterNode.Labels["cloud.google.com/gke-accelerator"],
			InstanceType: clusterNode.Labels["beta.kubernetes.io/instance-type"],
			Workloads:    []Workload{},
		}
	}

	return nodes, nil
}

func ListPods(ctx context.Context, client kubernetes.Interface) (*v1.PodList, error) {
	pods, err := client.CoreV1().Pods("").List(
		ctx,
		metav1.ListOptions{FieldSelector: "status.phase=Running,metadata.namespace!=kube-system,metadata.namespace!=gke-gmp-system,metadata.namespace!=gmp-system"},
	)
	if err != nil {
		return nil, fmt.Errorf("error getting pods: %w", err)
	}
	return pods, nil
}

func ListNamespaces(ctx context.Context, client kubernetes.Interface) (*v1.NamespaceList, error) {
	namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting namespaces: %w", err)
	}
	return namespaces, nil
}

func ListNodes(ctx context.Context, client kubernetes.Interface) (*v1.NodeList, error) {
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting nodes: %w", err)
	}
	return nodes, nil
}

func DescribePod(ctx context.Context, client kubernetes.Interface, podName string, namespace string) (*v1.Pod, error) {
	pod, err := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting pod %s in namespace %s: %w", podName, namespace, err)
	}
	return pod, nil
}
