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

package calculator

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/autopilot-cost-calculator/cluster"
	"gopkg.in/ini.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

const (
	CLUSTER_FEE   = 0.1
	HOURS_PER_MONTH = 730.0
)

type PricingService struct {
	AutopilotPricing AutopilotPriceList
	GCEPricing       GCEPriceList
	Config           *ini.File
	clientset        kubernetes.Interface
	metricsClientset metricsv.Interface
}

type CostSummary struct {
	TotalWorkloads     int                    `json:"total_workloads"`
	TotalNodes         int                    `json:"total_nodes"`
	TotalCpuMcpu       int64                  `json:"total_cpu_mcpu"`
	TotalMemoryMib     int64                  `json:"total_memory_mib"`
	TotalStorageMib    int64                  `json:"total_storage_mib"`
	TotalGpus          int64                  `json:"total_gpus"`
	SpotWorkloadsCount int                    `json:"spot_workloads_count"`
	ClassDistribution  map[string]int         `json:"class_distribution"`

	// Hourly costs
	HourlyWorkloadsCost float64 `json:"hourly_workloads_cost"`
	HourlySpotCost      float64 `json:"hourly_spot_cost"`
	HourlyNonSpotCost   float64 `json:"hourly_non_spot_cost"`
	HourlyClusterFee    float64 `json:"hourly_cluster_fee"`
	HourlyTotalOnDemand float64 `json:"hourly_total_on_demand"`
	Hourly1YearCommit   float64 `json:"hourly_1year_commit"`
	Hourly3YearCommit   float64 `json:"hourly_3year_commit"`

	// Monthly costs (730 hours)
	MonthlyWorkloadsCost float64 `json:"monthly_workloads_cost"`
	MonthlySpotCost      float64 `json:"monthly_spot_cost"`
	MonthlyNonSpotCost   float64 `json:"monthly_non_spot_cost"`
	MonthlyClusterFee    float64 `json:"monthly_cluster_fee"`
	MonthlyTotalOnDemand float64 `json:"monthly_total_on_demand"`
	Monthly1YearCommit   float64 `json:"monthly_1year_commit"`
	Monthly3YearCommit   float64 `json:"monthly_3year_commit"`

	// Savings
	Savings1YearMonthly    float64 `json:"savings_1year_monthly"`
	Savings1YearPercentage float64 `json:"savings_1year_percentage"`
	Savings3YearMonthly    float64 `json:"savings_3year_monthly"`
	Savings3YearPercentage float64 `json:"savings_3year_percentage"`
}

func NewService(ctx context.Context, sku map[string]string, region string, clientset kubernetes.Interface, metricsClientset metricsv.Interface, config *ini.File) (*PricingService, error) {
	apPricing, err := GetAutopilotPricing(ctx, sku["autopilot"], region)
	if err != nil {
		return nil, fmt.Errorf("failed to get Autopilot pricing: %w", err)
	}

	gcePricing, err := GetGCEPricing(ctx, sku["gce"], region)
	if err != nil {
		return nil, fmt.Errorf("failed to get GCE pricing: %w", err)
	}

	service := &PricingService{
		AutopilotPricing: apPricing,
		GCEPricing:       gcePricing,
		clientset:        clientset,
		metricsClientset: metricsClientset,
		Config:           config,
	}

	return service, nil
}

func NewMockService(region string, config *ini.File) *PricingService {
	apPricing, gcePricing := GetMockPricing(region)
	return &PricingService{
		AutopilotPricing: apPricing,
		GCEPricing:       gcePricing,
		Config:           config,
	}
}

func (service *PricingService) CalculatePricing(cpu int64, memory int64, storage int64, gpu int64, gpuModel string, class cluster.ComputeClass, instanceType string, spot bool) float64 {
	// If spot, calculations are done based on spot pricing
	if spot {
		switch class {
		case cluster.ComputeClassPerformance:
			perfPrice := service.AutopilotPricing.SpotPerformanceCpuPricePremium*float64(cpu)/1000 +
				service.AutopilotPricing.SpotPerformanceMemoryPricePremium*float64(memory)/1000 +
				service.AutopilotPricing.SpotPerformanceLocalSSDPricePremium*float64(storage)/1000

			gcePrice, _ := service.GetGCEMachinePrice(instanceType, spot)
			return perfPrice + gcePrice

		case cluster.ComputeClassAccelerator:
			acceleratorPrice := service.AutopilotPricing.SpotAcceleratorCpuPricePremium*float64(cpu)/1000 +
				service.AutopilotPricing.SpotAcceleratorMemoryGPUPricePremium*float64(memory)/1000 +
				service.AutopilotPricing.SpotAcceleratorLocalSSDPricePremium*float64(storage)/1000

			switch gpuModel {
			case "nvidia-tesla-t4":
				acceleratorPrice += service.AutopilotPricing.SpotAcceleratorT4GPUPricePremium * float64(gpu)
			case "nvidia-l4":
				acceleratorPrice += service.AutopilotPricing.SpotAcceleratorL4GPUPricePremium * float64(gpu)
			case "nvidia-tesla-a100":
				acceleratorPrice += service.AutopilotPricing.SpotAcceleratorA10040GGPUPricePremium * float64(gpu)
			case "nvidia-a100-80gb":
				acceleratorPrice += service.AutopilotPricing.SpotAcceleratorA10080GGPUPricePremium * float64(gpu)
			case "nvidia-h100-80gb":
				acceleratorPrice += service.AutopilotPricing.SpotAcceleratorH100GPUPricePremium * float64(gpu)
			}

			gcePrice, _ := service.GetGCEMachinePrice(instanceType, spot)
			return acceleratorPrice + gcePrice

		case cluster.ComputeClassGPUPod:
			acceleratorPrice := service.AutopilotPricing.SpotGPUPodvCPUPrice*float64(cpu)/1000 +
				service.AutopilotPricing.SpotGPUPodMemoryPrice*float64(memory)/1000 +
				service.AutopilotPricing.SpotGPUPodLocalSSDPrice*float64(storage)/1000

			switch gpuModel {
			case "nvidia-tesla-t4":
				acceleratorPrice += service.AutopilotPricing.SpotNVIDIAT4PodGPUPrice * float64(gpu)
			case "nvidia-l4":
				acceleratorPrice += service.AutopilotPricing.SpotNVIDIAL4PodGPUPrice * float64(gpu)
			case "nvidia-tesla-a100":
				acceleratorPrice += service.AutopilotPricing.SpotNVIDIAA10040GPodGPUPrice * float64(gpu)
			case "nvidia-a100-80gb":
				acceleratorPrice += service.AutopilotPricing.SpotNVIDIAA10080GPodGPUPrice * float64(gpu)
			}
			return acceleratorPrice

		case cluster.ComputeClassBalanced:
			return service.AutopilotPricing.SpotCpuBalancedPrice*float64(cpu)/1000 +
				service.AutopilotPricing.SpotMemoryBalancedPrice*float64(memory)/1000 +
				service.AutopilotPricing.StoragePrice*float64(storage)/1000

		case cluster.ComputeClassScaleout:
			return service.AutopilotPricing.SpotCpuScaleoutPrice*float64(cpu)/1000 +
				service.AutopilotPricing.SpotMemoryScaleoutPrice*float64(memory)/1000 +
				service.AutopilotPricing.StoragePrice*float64(storage)/1000

		case cluster.ComputeClassScaleoutArm:
			return service.AutopilotPricing.SpotArmCpuScaleoutPrice*float64(cpu)/1000 +
				service.AutopilotPricing.SpotArmMemoryScaleoutPrice*float64(memory)/1000 +
				service.AutopilotPricing.StoragePrice*float64(storage)/1000

		default:
			return service.AutopilotPricing.SpotCpuPrice*float64(cpu)/1000 +
				service.AutopilotPricing.SpotMemoryPrice*float64(memory)/1000 +
				service.AutopilotPricing.StoragePrice*float64(storage)/1000
		}
	}

	switch class {
	case cluster.ComputeClassPerformance:
		perfPrice := service.AutopilotPricing.PerformanceCpuPricePremium*float64(cpu)/1000 +
			service.AutopilotPricing.PerformanceMemoryPricePremium*float64(memory)/1000 +
			service.AutopilotPricing.PerformanceLocalSSDPricePremium*float64(storage)/1000

		gcePrice, _ := service.GetGCEMachinePrice(instanceType, spot)
		return perfPrice + gcePrice

	case cluster.ComputeClassAccelerator:
		acceleratorPrice := service.AutopilotPricing.AcceleratorCpuPricePremium*float64(cpu)/1000 +
			service.AutopilotPricing.AcceleratorMemoryGPUPricePremium*float64(memory)/1000 +
			service.AutopilotPricing.AcceleratorLocalSSDPricePremium*float64(storage)/1000

		switch gpuModel {
		case "nvidia-tesla-t4":
			acceleratorPrice += service.AutopilotPricing.AcceleratorT4GPUPricePremium * float64(gpu)
		case "nvidia-l4":
			acceleratorPrice += service.AutopilotPricing.AcceleratorL4GPUPricePremium * float64(gpu)
		case "nvidia-tesla-a100":
			acceleratorPrice += service.AutopilotPricing.AcceleratorA10040GGPUPricePremium * float64(gpu)
		case "nvidia-a100-80gb":
			acceleratorPrice += service.AutopilotPricing.AcceleratorA10080GGPUPricePremium * float64(gpu)
		case "nvidia-h100-80gb":
			acceleratorPrice += service.AutopilotPricing.AcceleratorH100GPUPricePremium * float64(gpu)
		}

		gcePrice, _ := service.GetGCEMachinePrice(instanceType, spot)
		return acceleratorPrice + gcePrice

	case cluster.ComputeClassGPUPod:
		acceleratorPrice := service.AutopilotPricing.GPUPodvCPUPrice*float64(cpu)/1000 +
			service.AutopilotPricing.GPUPodMemoryPrice*float64(memory)/1000 +
			service.AutopilotPricing.GPUPodLocalSSDPrice*float64(storage)/1000

		switch gpuModel {
		case "nvidia-tesla-t4":
			acceleratorPrice += service.AutopilotPricing.NVIDIAT4PodGPUPrice * float64(gpu)
		case "nvidia-l4":
			acceleratorPrice += service.AutopilotPricing.NVIDIAL4PodGPUPrice * float64(gpu)
		case "nvidia-tesla-a100":
			acceleratorPrice += service.AutopilotPricing.NVIDIAA10040GPodGPUPrice * float64(gpu)
		case "nvidia-a100-80gb":
			acceleratorPrice += service.AutopilotPricing.NVIDIAA10080GPodGPUPrice * float64(gpu)
		}
		return acceleratorPrice

	case cluster.ComputeClassBalanced:
		return service.AutopilotPricing.CpuBalancedPrice*float64(cpu)/1000 +
			service.AutopilotPricing.MemoryBalancedPrice*float64(memory)/1000 +
			service.AutopilotPricing.StoragePrice*float64(storage)/1000

	case cluster.ComputeClassScaleout:
		return service.AutopilotPricing.CpuScaleoutPrice*float64(cpu)/1000 +
			service.AutopilotPricing.MemoryScaleoutPrice*float64(memory)/1000 +
			service.AutopilotPricing.StoragePrice*float64(storage)/1000

	case cluster.ComputeClassScaleoutArm:
		return service.AutopilotPricing.CpuArmScaleoutPrice*float64(cpu)/1000 +
			service.AutopilotPricing.MemoryArmScaleoutPrice*float64(memory)/1000 +
			service.AutopilotPricing.StoragePrice*float64(storage)/1000

	default:
		return service.AutopilotPricing.CpuPrice*float64(cpu)/1000 +
			service.AutopilotPricing.MemoryPrice*float64(memory)/1000 +
			service.AutopilotPricing.StoragePrice*float64(storage)/1000
	}
}

func (service *PricingService) GetGCEMachinePrice(instanceType string, spot bool) (float64, error) {
	instanceInfo := strings.Split(instanceType, "-")
	if len(instanceInfo) < 3 {
		return 0, nil
	}

	cpus, err := strconv.Atoi(instanceInfo[2])
	if err != nil {
		return 0, nil
	}

	ram := 0.0
	classType := instanceInfo[1]
	machineType := instanceInfo[0]

	switch classType {
	case "standard":
		ram = float64(cpus) * 4
	case "highcpu":
		ram = float64(cpus) * 2
	case "highmem":
		ram = float64(cpus) * 4
	case "highgpu":
		ram = float64(cpus) * 7.0833
	case "ultragpu":
		ram = float64(cpus) * 14.1666
	}

	ram = math.Ceil(ram)

	if spot {
		switch machineType {
		case "a2":
			return service.GCEPricing.SpotA2CpuPrice*float64(cpus) + service.GCEPricing.SpotA2MemoryPrice*ram, nil
		case "a3":
			return service.GCEPricing.SpotA3CpuPrice*float64(cpus) + service.GCEPricing.SpotA3MemoryPrice*ram, nil
		case "g2":
			return service.GCEPricing.SpotG2DCpuPrice*float64(cpus) + service.GCEPricing.SpotG2DMemoryPrice*ram, nil
		case "h3":
			return service.GCEPricing.H3CpuPrice*float64(cpus) + service.GCEPricing.H3MemoryPrice*ram, nil
		case "c2":
			return service.GCEPricing.SpotC2CpuPrice*float64(cpus) + service.GCEPricing.SpotC2MemoryPrice*ram, nil
		case "c2d":
			return service.GCEPricing.SpotC2DCpuPrice*float64(cpus) + service.GCEPricing.SpotC2DMemoryPrice*ram, nil
		default:
			return 0, nil
		}
	}

	switch machineType {
	case "a2":
		return service.GCEPricing.A2CpuPrice*float64(cpus) + service.GCEPricing.A2MemoryPrice*ram, nil
	case "a3":
		return service.GCEPricing.A3CpuPrice*float64(cpus) + service.GCEPricing.A3MemoryPrice*ram, nil
	case "g2":
		return service.GCEPricing.G2CpuPrice*float64(cpus) + service.GCEPricing.G2MemoryPrice*ram, nil
	case "h3":
		return service.GCEPricing.H3CpuPrice*float64(cpus) + service.GCEPricing.H3MemoryPrice*ram, nil
	case "c2":
		return service.GCEPricing.C2CpuPrice*float64(cpus) + service.GCEPricing.C2MemoryPrice*ram, nil
	case "c2d":
		return service.GCEPricing.C2DCpuPrice*float64(cpus) + service.GCEPricing.C2DMemoryPrice*ram, nil
	default:
		return 0, nil
	}
}

func (service *PricingService) PopulateWorkloads(ctx context.Context, nodes map[string]cluster.Node, namespaceFilter string) ([]cluster.Workload, error) {
	var workloads []cluster.Workload

	podMetricsList, err := service.metricsClientset.MetricsV1beta1().PodMetricses(namespaceFilter).List(
		ctx,
		metav1.ListOptions{FieldSelector: "metadata.namespace!=kube-system,metadata.namespace!=gke-gmp-system,metadata.namespace!=gmp-system"},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pod metrics: %w", err)
	}

	for _, v := range podMetricsList.Items {
		pod, err := cluster.DescribePod(ctx, service.clientset, v.Name, v.Namespace)
		if err != nil {
			return nil, err
		}

		var cpu int64 = 0
		var memory int64 = 0
		var storage int64 = 0
		var gpu int64 = 0
		podContainerCount := 0

		gpuModel := pod.Spec.NodeSelector["cloud.google.com/gke-accelerator"]

		// Sum used resources from the Pod
		for _, container := range v.Containers {
			cpuUsage := container.Usage.Cpu().MilliValue()
			memoryUsage := container.Usage.Memory().MilliValue() / 1000000000            // Division to get MiB
			storageUsage := container.Usage.StorageEphemeral().MilliValue() / 1000000000 // Division to get MiB
			gpuUsage := int64(0)

			for _, specContainer := range pod.Spec.Containers {
				if container.Name == specContainer.Name {
					cpuRequest := specContainer.Resources.Requests[corev1.ResourceCPU]
					memoryRequest := specContainer.Resources.Requests[corev1.ResourceMemory]
					storageRequest := specContainer.Resources.Requests[corev1.ResourceStorage]
					gpuRequests := specContainer.Resources.Requests["nvidia.com/gpu"]

					// Usage is less than requests, so we set request as usage since billing allocates requests
					if cpuUsage < cpuRequest.MilliValue() {
						cpuUsage = cpuRequest.MilliValue()
					}

					if memoryUsage < memoryRequest.MilliValue()/1000000000 {
						memoryUsage = memoryRequest.MilliValue() / 1000000000
					}

					if storageUsage < storageRequest.MilliValue()/1000000000 {
						storageUsage = storageRequest.MilliValue() / 1000000000
					}

					gpuUsage = gpuRequests.Value()
				}
			}

			cpu += cpuUsage
			memory += memoryUsage
			storage += storageUsage
			gpu += gpuUsage
			podContainerCount++
		}

		// Check and modify the limits of summed workloads from the Pod
		cpu, memory, storage = ValidateAndRoundResources(cpu, memory, storage)

		nodeInfo := nodes[pod.Spec.NodeName]
		computeClass := service.DecideComputeClass(
			v.Name,
			nodeInfo.InstanceType,
			cpu,
			memory,
			gpu,
			gpuModel,
			strings.Contains(nodeInfo.InstanceType, service.Config.Section("").Key("gce_arm64_prefix").String()),
		)

		cost := service.CalculatePricing(cpu, memory, storage, gpu, gpuModel, computeClass, nodeInfo.InstanceType, nodeInfo.Spot)

		workloadObject := cluster.Workload{
			Name:              v.Name,
			Namespace:         v.Namespace,
			Containers:        podContainerCount,
			NodeName:          pod.Spec.NodeName,
			Cpu:               cpu,
			Memory:            memory,
			Storage:           storage,
			AcceleratorType:   gpuModel,
			AcceleratorAmount: gpu,
			Cost:              cost,
			ComputeClass:      computeClass,
		}

		workloads = append(workloads, workloadObject)

		if entry, ok := nodes[pod.Spec.NodeName]; ok {
			entry.Workloads = append(entry.Workloads, workloadObject)
			entry.Cost += cost
			nodes[pod.Spec.NodeName] = entry
		}
	}

	return workloads, nil
}

func (service *PricingService) DecideComputeClass(workloadName string, machineType string, mCPU int64, memory int64, gpu int64, gpuModel string, arm64 bool) cluster.ComputeClass {
	ratio := math.Ceil(float64(memory) / float64(mCPU))

	ratioRegularMin, _ := service.Config.Section("ratios").Key("generalpurpose_min").Float64()
	ratioRegularMax, _ := service.Config.Section("ratios").Key("generalpurpose_max").Float64()
	ratioBalancedMin, _ := service.Config.Section("ratios").Key("balanced_min").Float64()
	ratioBalancedMax, _ := service.Config.Section("ratios").Key("balanced_max").Float64()
	ratioScaleoutMin, _ := service.Config.Section("ratios").Key("scaleout_min").Float64()
	ratioScaleoutMax, _ := service.Config.Section("ratios").Key("scaleout_max").Float64()
	ratioPerformanceMin, _ := service.Config.Section("ratios").Key("performance_min").Float64()
	ratioPerformanceMax, _ := service.Config.Section("ratios").Key("performance_max").Float64()

	scaleoutMcpuMax, _ := service.Config.Section("limits").Key("scaleout_mcpu_max").Int64()
	scaleoutMemoryMax, _ := service.Config.Section("limits").Key("scaleout_memory_max").Int64()
	scaleoutArmMcpuMax, _ := service.Config.Section("limits").Key("scaleout_arm_mcpu_max").Int64()
	scaleoutArmMemoryMax, _ := service.Config.Section("limits").Key("scaleout_arm_memory_max").Int64()
	regularMcpuMax, _ := service.Config.Section("limits").Key("generalpurpose_mcpu_max").Int64()
	regularMemoryMax, _ := service.Config.Section("limits").Key("generalpurpose_memory_max").Int64()
	balancedMcpuMax, _ := service.Config.Section("limits").Key("balanced_mcpu_max").Int64()
	balancedMemoryMax, _ := service.Config.Section("limits").Key("balanced_memory_max").Int64()
	performanceMcpuMax, _ := service.Config.Section("limits").Key("performance_mcpu_max").Int64()
	performanceMemoryMax, _ := service.Config.Section("limits").Key("performance_memory_max").Int64()

	gpupodT4McpuMin, _ := service.Config.Section("limits").Key("gpupod_t4_mcpu_min").Int64()
	gpupodT4McpuMax, _ := service.Config.Section("limits").Key("gpupod_t4_mcpu_max").Int64()
	gpupodT4MemoryMin, _ := service.Config.Section("limits").Key("gpupod_t4_memory_min").Int64()
	gpupodT4MemoryMax, _ := service.Config.Section("limits").Key("gpupod_t4_memory_max").Int64()

	gpupodL4McpuMin, _ := service.Config.Section("limits").Key("gpupod_l4_mcpu_min").Int64()
	gpupodL4McpuMax, _ := service.Config.Section("limits").Key("gpupod_l4_mcpu_max").Int64()
	gpupodL4MemoryMin, _ := service.Config.Section("limits").Key("gpupod_l4_memory_min").Int64()
	gpupodL4MemoryMax, _ := service.Config.Section("limits").Key("gpupod_l4_memory_max").Int64()

	gpupodA10040McpuMin, _ := service.Config.Section("limits").Key("gpupod_a100_40_mcpu_min").Int64()
	gpupodA10040McpuMax, _ := service.Config.Section("limits").Key("gpupod_a100_40_mcpu_max").Int64()
	gpupodA10040MemoryMin, _ := service.Config.Section("limits").Key("gpupod_a100_40_memory_min").Int64()
	gpupodA10040MemoryMax, _ := service.Config.Section("limits").Key("gpupod_a100_40_memory_max").Int64()

	gpupodA10080McpuMin, _ := service.Config.Section("limits").Key("gpupod_a100_80_mcpu_min").Int64()
	gpupodA10080McpuMax, _ := service.Config.Section("limits").Key("gpupod_a100_80_mcpu_max").Int64()
	gpupodA10080MemoryMin, _ := service.Config.Section("limits").Key("gpupod_a100_80_memory_min").Int64()
	gpupodA10080MemoryMax, _ := service.Config.Section("limits").Key("gpupod_a100_80_memory_max").Int64()

	acceleratorMcpuMin, _ := service.Config.Section("limits").Key("accelerator_mcpu_min").Int64()
	acceleratorMemoryMin, _ := service.Config.Section("limits").Key("accelerator_memory_min").Int64()
	acceleratorH10080McpuMax, _ := service.Config.Section("limits").Key("accelerator_h100_80_mcpu_max").Int64()
	acceleratorH10080MemoryMax, _ := service.Config.Section("limits").Key("accelerator_h100_80_memory_max").Int64()

	computeOptimizedMachineTypes := strings.Split(service.Config.Section("").Key("gce_compute_optimized_prefixed").String(), ",")
	for _, computeOptimizedMachineType := range computeOptimizedMachineTypes {
		if computeOptimizedMachineType != "" && strings.Contains(machineType, computeOptimizedMachineType) {
			return cluster.ComputeClassPerformance
		}
	}

	// check if GPU is H100, then return ComputeClassAccelerator since it's the only one supporting these GPUs
	if gpuModel == service.Config.Section("").Key("nvidia_h100_identifier").String() {
		if ratio < ratioPerformanceMin || ratio > ratioPerformanceMax || mCPU > performanceMcpuMax || memory > performanceMemoryMax {
			log.Printf("Warning: Requested memory or CPU out of recommended range for Performance compute class (%s) workload (%s).\n", machineType, workloadName)
		}
		return cluster.ComputeClassPerformance
	}

	acceleratorOptimizedMachineTypes := strings.Split(service.Config.Section("").Key("gce_accelerator_optimized_prefixed").String(), ",")
	for _, acceleratorOptimizedMachineType := range acceleratorOptimizedMachineTypes {
		if acceleratorOptimizedMachineType != "" && strings.Contains(machineType, acceleratorOptimizedMachineType) {
			switch gpuModel {
			case "nvidia-tesla-t4":
				if mCPU > gpupodT4McpuMax || mCPU < acceleratorMcpuMin || memory > gpupodT4MemoryMax || memory < acceleratorMemoryMin {
					log.Printf("Warning: Requested memory or CPU out of recommended range for %s Accelerator compute class (%s) workload (%s).\n", machineType, gpuModel, workloadName)
				}
			case "nvidia-l4":
				if mCPU > gpupodL4McpuMax || mCPU < acceleratorMcpuMin || memory > gpupodL4MemoryMax || memory < acceleratorMemoryMin {
					log.Printf("Warning: Requested memory or CPU out of recommended range for %s Accelerator compute class (%s) workload (%s).\n", machineType, gpuModel, workloadName)
				}
			case "nvidia-tesla-a100":
				if mCPU > gpupodA10040McpuMax || mCPU < acceleratorMcpuMin || memory > gpupodA10040MemoryMax || memory < acceleratorMemoryMin {
					log.Printf("Warning: Requested memory or CPU out of recommended range for %s Accelerator compute class (%s) workload (%s).\n", machineType, gpuModel, workloadName)
				}
			case "nvidia-a100-80gb":
				if mCPU > gpupodA10080McpuMax || mCPU < acceleratorMcpuMin || memory > gpupodA10080MemoryMax || memory < acceleratorMemoryMin {
					log.Printf("Warning: Requested memory or CPU out of recommended range for %s Accelerator compute class (%s) workload (%s).\n", machineType, gpuModel, workloadName)
				}
			case "nvidia-h100-80gb":
				if mCPU > acceleratorH10080McpuMax || mCPU < acceleratorMcpuMin || memory > acceleratorH10080MemoryMax || memory < acceleratorMemoryMin {
					log.Printf("Warning: Requested memory or CPU out of recommended range for %s Accelerator compute class (%s) workload (%s).\n", machineType, gpuModel, workloadName)
				}
			}

			return cluster.ComputeClassAccelerator
		}
	}

	// GPU Pod type
	if gpu > 0 {
		switch gpuModel {
		case "nvidia-tesla-t4":
			if mCPU > gpupodT4McpuMax || mCPU < gpupodT4McpuMin || memory > gpupodT4MemoryMax || memory < gpupodT4MemoryMin {
				log.Printf("Warning: Requested memory or CPU out of recommended range for %s GPU workload (%s).\n", gpuModel, workloadName)
			}
		case "nvidia-l4":
			if mCPU > gpupodL4McpuMax || mCPU < gpupodL4McpuMin || memory > gpupodL4MemoryMax || memory < gpupodL4MemoryMin {
				log.Printf("Warning: Requested memory or CPU out of recommended range for %s GPU workload (%s).\n", gpuModel, workloadName)
			}
		case "nvidia-tesla-a100":
			if mCPU > gpupodA10040McpuMax || mCPU < gpupodA10040McpuMin || memory > gpupodA10040MemoryMax || memory < gpupodA10040MemoryMin {
				log.Printf("Warning: Requested memory or CPU out of recommended range for %s GPU workload (%s).\n", gpuModel, workloadName)
			}
		case "nvidia-a100-80gb":
			if mCPU > gpupodA10080McpuMax || mCPU < gpupodA10080McpuMin || memory > gpupodA10080MemoryMax || memory < gpupodA10080MemoryMin {
				log.Printf("Warning: Requested memory or CPU out of recommended range for %s GPU workload (%s).\n", gpuModel, workloadName)
			}
		}
		return cluster.ComputeClassGPUPod
	}

	// ARM64 Scale-out
	if arm64 {
		if ratio < ratioScaleoutMin || ratio > ratioScaleoutMax || mCPU > scaleoutArmMcpuMax || memory > scaleoutArmMemoryMax {
			log.Printf("Warning: ARM64 workload %s resource ratios out of standard range.\n", workloadName)
		}
		return cluster.ComputeClassScaleoutArm
	}

	// Standard ranges
	if ratio >= ratioRegularMin && ratio <= ratioRegularMax && mCPU <= regularMcpuMax && memory <= regularMemoryMax {
		return cluster.ComputeClassGeneralPurpose
	}

	if ratio >= ratioScaleoutMin && ratio <= ratioScaleoutMax && mCPU <= scaleoutMcpuMax && memory <= scaleoutMemoryMax {
		return cluster.ComputeClassScaleout
	}

	if ratio >= ratioBalancedMin && ratio <= ratioBalancedMax && mCPU <= balancedMcpuMax && memory <= balancedMemoryMax {
		return cluster.ComputeClassBalanced
	}

	return cluster.ComputeClassGeneralPurpose
}

func ValidateAndRoundResources(mCPU int64, memory int64, storage int64) (int64, int64, int64) {
	// Lowest possible mCPU request in Autopilot
	if mCPU < 50 {
		mCPU = 50
	}

	// Minimum memory request
	if memory < 52 {
		memory = 52
	}

	// Minimum ephemeral storage request
	if storage < 10 {
		storage = 10
	}

	// Round to nearest 50 mCPU step for granular accounting
	mCPUMissing := (50 - (mCPU % 50))
	if mCPUMissing != 50 {
		mCPU += mCPUMissing
	}

	return mCPU, memory, storage
}

func (service *PricingService) CalculateSummary(nodes map[string]cluster.Node, workloads []cluster.Workload, oneYearDiscount float64, threeYearDiscount float64, clusterFee float64) CostSummary {
	var summary CostSummary
	summary.TotalNodes = len(nodes)
	summary.TotalWorkloads = len(workloads)
	summary.HourlyClusterFee = clusterFee
	summary.MonthlyClusterFee = clusterFee * HOURS_PER_MONTH
	summary.ClassDistribution = make(map[string]int)

	for _, w := range workloads {
		summary.TotalCpuMcpu += w.Cpu
		summary.TotalMemoryMib += w.Memory
		summary.TotalStorageMib += w.Storage
		summary.TotalGpus += w.AcceleratorAmount
		summary.ClassDistribution[w.ComputeClass.String()]++
	}

	for _, node := range nodes {
		for _, w := range node.Workloads {
			if node.Spot {
				summary.HourlySpotCost += w.Cost
				summary.SpotWorkloadsCount++
			} else {
				summary.HourlyNonSpotCost += w.Cost
			}
		}
	}

	summary.HourlyWorkloadsCost = summary.HourlySpotCost + summary.HourlyNonSpotCost
	summary.HourlyTotalOnDemand = summary.HourlyWorkloadsCost + summary.HourlyClusterFee

	summary.Hourly1YearCommit = (summary.HourlySpotCost + summary.HourlyNonSpotCost*oneYearDiscount) + summary.HourlyClusterFee
	summary.Hourly3YearCommit = (summary.HourlySpotCost + summary.HourlyNonSpotCost*threeYearDiscount) + summary.HourlyClusterFee

	summary.MonthlyWorkloadsCost = summary.HourlyWorkloadsCost * HOURS_PER_MONTH
	summary.MonthlySpotCost = summary.HourlySpotCost * HOURS_PER_MONTH
	summary.MonthlyNonSpotCost = summary.HourlyNonSpotCost * HOURS_PER_MONTH
	summary.MonthlyTotalOnDemand = summary.HourlyTotalOnDemand * HOURS_PER_MONTH
	summary.Monthly1YearCommit = summary.Hourly1YearCommit * HOURS_PER_MONTH
	summary.Monthly3YearCommit = summary.Hourly3YearCommit * HOURS_PER_MONTH

	summary.Savings1YearMonthly = summary.MonthlyTotalOnDemand - summary.Monthly1YearCommit
	if summary.MonthlyTotalOnDemand > 0 {
		summary.Savings1YearPercentage = (summary.Savings1YearMonthly / summary.MonthlyTotalOnDemand) * 100
	}

	summary.Savings3YearMonthly = summary.MonthlyTotalOnDemand - summary.Monthly3YearCommit
	if summary.MonthlyTotalOnDemand > 0 {
		summary.Savings3YearPercentage = (summary.Savings3YearMonthly / summary.MonthlyTotalOnDemand) * 100
	}

	return summary
}
