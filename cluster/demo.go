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

// GetDemoClusterData generates representative sample cluster data for testing and demonstration
func GetDemoClusterData() (ClusterInfo, map[string]Node) {
	info := ClusterInfo{
		Name:          "prod-ecommerce-cluster",
		Project:       "my-gcp-production-project",
		Location:      "us-central1",
		Status:        "RUNNING",
		MasterVersion: "1.31.2-gke.1200",
		NodeCount:     4,
		WorkloadCount: 8,
	}

	nodes := map[string]Node{
		"gke-prod-cluster-standard-pool-1a2b": {
			Name:         "gke-prod-cluster-standard-pool-1a2b",
			InstanceType: "e2-standard-4",
			Region:       "us-central1",
			Spot:         false,
			Accelerator:  "",
			Workloads: []Workload{
				{
					Name:            "frontend-web-deployment-78f4",
					Namespace:       "frontend",
					NodeName:        "gke-prod-cluster-standard-pool-1a2b",
					Containers:      2,
					Cpu:             500,
					Memory:          1024,
					Storage:         1024,
					ComputeClass:    ComputeClassGeneralPurpose,
					Cost:            0.035198,
				},
				{
					Name:            "api-gateway-service-5d6c",
					Namespace:       "api",
					NodeName:        "gke-prod-cluster-standard-pool-1a2b",
					Containers:      1,
					Cpu:             1000,
					Memory:          2048,
					Storage:         2048,
					ComputeClass:    ComputeClassGeneralPurpose,
					Cost:            0.070396,
				},
			},
			Cost: 0.105594,
		},
		"gke-prod-cluster-standard-pool-3c4d": {
			Name:         "gke-prod-cluster-standard-pool-3c4d",
			InstanceType: "c2-standard-8",
			Region:       "us-central1",
			Spot:         false,
			Accelerator:  "",
			Workloads: []Workload{
				{
					Name:            "order-processing-engine-9e8a",
					Namespace:       "core",
					NodeName:        "gke-prod-cluster-standard-pool-3c4d",
					Containers:      3,
					Cpu:             4000,
					Memory:          16384,
					Storage:         5120,
					ComputeClass:    ComputeClassPerformance,
					Cost:            0.392764,
				},
				{
					Name:            "redis-inmemory-cache-0",
					Namespace:       "cache",
					NodeName:        "gke-prod-cluster-standard-pool-3c4d",
					Containers:      1,
					Cpu:             2000,
					Memory:          8192,
					Storage:         2048,
					ComputeClass:    ComputeClassBalanced,
					Cost:            0.241748,
				},
			},
			Cost: 0.634512,
		},
		"gke-prod-cluster-spot-pool-5e6f": {
			Name:         "gke-prod-cluster-spot-pool-5e6f",
			InstanceType: "t2a-standard-4",
			Region:       "us-central1",
			Spot:         true,
			Accelerator:  "",
			Workloads: []Workload{
				{
					Name:            "analytics-etl-batch-runner-1b",
					Namespace:       "analytics",
					NodeName:        "gke-prod-cluster-spot-pool-5e6f",
					Containers:      2,
					Cpu:             2000,
					Memory:          8192,
					Storage:         10240,
					ComputeClass:    ComputeClassScaleoutArm,
					Cost:            0.045610,
				},
				{
					Name:            "image-resizer-worker-99x2",
					Namespace:       "media",
					NodeName:        "gke-prod-cluster-spot-pool-5e6f",
					Containers:      1,
					Cpu:             1000,
					Memory:          4096,
					Storage:         2048,
					ComputeClass:    ComputeClassScaleoutArm,
					Cost:            0.022805,
				},
			},
			Cost: 0.068415,
		},
		"gke-prod-cluster-gpu-pool-7g8h": {
			Name:         "gke-prod-cluster-gpu-pool-7g8h",
			InstanceType: "g2-standard-4",
			Region:       "us-central1",
			Spot:         false,
			Accelerator:  "nvidia-l4",
			Workloads: []Workload{
				{
					Name:              "llm-inference-serving-v2",
					Namespace:         "ml-models",
					NodeName:          "gke-prod-cluster-gpu-pool-7g8h",
					Containers:        2,
					Cpu:               4000,
					Memory:            16384,
					Storage:           20480,
					AcceleratorType:   "nvidia-l4",
					AcceleratorAmount: 1,
					ComputeClass:      ComputeClassGPUPod,
					Cost:              1.091560,
				},
				{
					Name:              "embeddings-indexer-job-44a",
					Namespace:         "ml-models",
					NodeName:          "gke-prod-cluster-gpu-pool-7g8h",
					Containers:        1,
					Cpu:               2000,
					Memory:            8192,
					Storage:           10240,
					AcceleratorType:   "nvidia-l4",
					AcceleratorAmount: 1,
					ComputeClass:      ComputeClassGPUPod,
					Cost:              0.884780,
				},
			},
			Cost: 1.976340,
		},
	}

	return info, nodes
}
