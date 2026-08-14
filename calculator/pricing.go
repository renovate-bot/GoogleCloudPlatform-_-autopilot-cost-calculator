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
	"slices"
	"strings"

	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/option"
)

type GCEPriceList struct {
	Region string `json:"region"`

	H3CpuPrice    float64 `json:"h3_cpu_price"`
	H3MemoryPrice float64 `json:"h3_memory_price"`

	C2CpuPrice     float64 `json:"c2_cpu_price"`
	C2MemoryPrice  float64 `json:"c2_memory_price"`
	C2DCpuPrice    float64 `json:"c2d_cpu_price"`
	C2DMemoryPrice float64 `json:"c2d_memory_price"`

	G2CpuPrice    float64 `json:"g2_cpu_price"`
	G2MemoryPrice float64 `json:"g2_memory_price"`
	A2CpuPrice    float64 `json:"a2_cpu_price"`
	A2MemoryPrice float64 `json:"a2_memory_price"`
	A3CpuPrice    float64 `json:"a3_cpu_price"`
	A3MemoryPrice float64 `json:"a3_memory_price"`

	SpotC2CpuPrice     float64 `json:"spot_c2_cpu_price"`
	SpotC2MemoryPrice  float64 `json:"spot_c2_memory_price"`
	SpotC2DCpuPrice    float64 `json:"spot_c2d_cpu_price"`
	SpotC2DMemoryPrice float64 `json:"spot_c2d_memory_price"`

	SpotG2DCpuPrice    float64 `json:"spot_g2d_cpu_price"`
	SpotG2DMemoryPrice float64 `json:"spot_g2d_memory_price"`
	SpotA2CpuPrice     float64 `json:"spot_a2_cpu_price"`
	SpotA2MemoryPrice  float64 `json:"spot_a2_memory_price"`
	SpotA3CpuPrice     float64 `json:"spot_a3_cpu_price"`
	SpotA3MemoryPrice  float64 `json:"spot_a3_memory_price"`
}

type AutopilotPriceList struct {
	Region       string  `json:"region"`
	StoragePrice float64 `json:"storage_price"`

	// Non-specific workloads (General purpose)
	CpuPrice        float64 `json:"cpu_price"`
	MemoryPrice     float64 `json:"memory_price"`
	SpotCpuPrice    float64 `json:"spot_cpu_price"`
	SpotMemoryPrice float64 `json:"spot_memory_price"`

	// Balanced workloads
	CpuBalancedPrice        float64 `json:"cpu_balanced_price"`
	MemoryBalancedPrice     float64 `json:"memory_balanced_price"`
	SpotCpuBalancedPrice    float64 `json:"spot_cpu_balanced_price"`
	SpotMemoryBalancedPrice float64 `json:"spot_memory_balanced_price"`

	// Scale-out workloads
	CpuScaleoutPrice        float64 `json:"cpu_scaleout_price"`
	MemoryScaleoutPrice     float64 `json:"memory_scaleout_price"`
	SpotCpuScaleoutPrice    float64 `json:"spot_cpu_scaleout_price"`
	SpotMemoryScaleoutPrice float64 `json:"spot_memory_scaleout_price"`

	// Scale-out ARM workloads
	CpuArmScaleoutPrice        float64 `json:"cpu_arm_scaleout_price"`
	MemoryArmScaleoutPrice     float64 `json:"memory_arm_scaleout_price"`
	SpotArmCpuScaleoutPrice    float64 `json:"spot_arm_cpu_scaleout_price"`
	SpotArmMemoryScaleoutPrice float64 `json:"spot_arm_memory_scaleout_price"`

	// GPU pricing
	GPUPodvCPUPrice              float64 `json:"gpu_pod_vcpu_price"`
	GPUPodMemoryPrice            float64 `json:"gpu_pod_memory_price"`
	GPUPodLocalSSDPrice          float64 `json:"gpu_pod_local_ssd_price"`
	NVIDIAL4PodGPUPrice          float64 `json:"nvidia_l4_pod_gpu_price"`
	NVIDIAT4PodGPUPrice          float64 `json:"nvidia_t4_pod_gpu_price"`
	NVIDIAA10040GPodGPUPrice     float64 `json:"nvidia_a100_40g_pod_gpu_price"`
	NVIDIAA10080GPodGPUPrice     float64 `json:"nvidia_a100_80g_pod_gpu_price"`
	SpotGPUPodvCPUPrice          float64 `json:"spot_gpu_pod_vcpu_price"`
	SpotGPUPodMemoryPrice        float64 `json:"spot_gpu_pod_memory_price"`
	SpotGPUPodLocalSSDPrice      float64 `json:"spot_gpu_pod_local_ssd_price"`
	SpotGPUPodPDPricePremium     float64 `json:"spot_gpu_pod_pd_price_premium"`
	SpotNVIDIAL4PodGPUPrice      float64 `json:"spot_nvidia_l4_pod_gpu_price"`
	SpotNVIDIAT4PodGPUPrice      float64 `json:"spot_nvidia_t4_pod_gpu_price"`
	SpotNVIDIAA10040GPodGPUPrice float64 `json:"spot_nvidia_a100_40g_pod_gpu_price"`
	SpotNVIDIAA10080GPodGPUPrice float64 `json:"spot_nvidia_a100_80g_pod_gpu_price"`

	// Performance tier baseline pricing
	PerformanceCpuPricePremium          float64 `json:"performance_cpu_price_premium"`
	PerformanceMemoryPricePremium       float64 `json:"performance_memory_price_premium"`
	PerformancePDPricePremium           float64 `json:"performance_pd_price_premium"`
	PerformanceLocalSSDPricePremium     float64 `json:"performance_local_ssd_price_premium"`
	SpotPerformanceCpuPricePremium      float64 `json:"spot_performance_cpu_price_premium"`
	SpotPerformanceMemoryPricePremium   float64 `json:"spot_performance_memory_price_premium"`
	SpotPerformancePDPricePremium       float64 `json:"spot_performance_pd_price_premium"`
	SpotPerformanceLocalSSDPricePremium float64 `json:"spot_performance_local_ssd_price_premium"`

	// Accelerator tier baseline pricing
	AcceleratorCpuPricePremium            float64 `json:"accelerator_cpu_price_premium"`
	AcceleratorMemoryGPUPricePremium      float64 `json:"accelerator_memory_gpu_price_premium"`
	AcceleratorPDPricePremium             float64 `json:"accelerator_pd_price_premium"`
	AcceleratorLocalSSDPricePremium       float64 `json:"accelerator_local_ssd_price_premium"`
	AcceleratorT4GPUPricePremium          float64 `json:"accelerator_t4_gpu_price_premium"`
	AcceleratorL4GPUPricePremium          float64 `json:"accelerator_l4_gpu_price_premium"`
	AcceleratorA10040GGPUPricePremium     float64 `json:"accelerator_a100_40g_gpu_price_premium"`
	AcceleratorA10080GGPUPricePremium     float64 `json:"accelerator_a100_80g_gpu_price_premium"`
	AcceleratorH100GPUPricePremium        float64 `json:"accelerator_h100_gpu_price_premium"`
	SpotAcceleratorCpuPricePremium        float64 `json:"spot_accelerator_cpu_price_premium"`
	SpotAcceleratorMemoryGPUPricePremium  float64 `json:"spot_accelerator_memory_gpu_price_premium"`
	SpotAcceleratorPDPricePremium         float64 `json:"spot_accelerator_pd_price_premium"`
	SpotAcceleratorLocalSSDPricePremium   float64 `json:"spot_accelerator_local_ssd_price_premium"`
	SpotAcceleratorT4GPUPricePremium      float64 `json:"spot_accelerator_t4_gpu_price_premium"`
	SpotAcceleratorL4GPUPricePremium      float64 `json:"spot_accelerator_l4_gpu_price_premium"`
	SpotAcceleratorA10040GGPUPricePremium float64 `json:"spot_accelerator_a100_40g_gpu_price_premium"`
	SpotAcceleratorA10080GGPUPricePremium float64 `json:"spot_accelerator_a100_80g_gpu_price_premium"`
	SpotAcceleratorH100GPUPricePremium    float64 `json:"spot_accelerator_h100_gpu_price_premium"`
}

func NormalizeRegion(region string) string {
	parts := strings.Split(region, "-")
	if len(parts) > 2 {
		return strings.Join(parts[:len(parts)-1], "-")
	}
	return region
}

func GetGCEPricing(ctx context.Context, sku string, region string) (GCEPriceList, error) {
	region = NormalizeRegion(region)
	pricing := GCEPriceList{Region: region}

	cloudbillingService, err := cloudbilling.NewService(ctx, option.WithScopes(cloudbilling.CloudPlatformScope))
	if err != nil {
		return GCEPriceList{}, fmt.Errorf("unable to initialize cloud billing service: %w", err)
	}

	err = cloudbillingService.Services.Skus.List("services/"+sku).CurrencyCode("USD").Pages(ctx, func(pricingInfo *cloudbilling.ListSkusResponse) error {
		for _, skuItem := range pricingInfo.Skus {
			if !slices.Contains(skuItem.ServiceRegions, region) {
				continue
			}
			if len(skuItem.PricingInfo) == 0 || len(skuItem.PricingInfo[0].PricingExpression.TieredRates) == 0 {
				continue
			}

			rate := skuItem.PricingInfo[0].PricingExpression.TieredRates[0]
			decimal := rate.UnitPrice.Units * 1000000000
			mantissa := rate.UnitPrice.Nanos * int64(skuItem.PricingInfo[0].PricingExpression.DisplayQuantity)
			price := float64(decimal+mantissa) / 1000000000

			switch {
			case strings.HasPrefix(skuItem.Description, "H3 Instance Core"):
				pricing.H3CpuPrice = price
			case strings.HasPrefix(skuItem.Description, "H3 Instance Ram"):
				pricing.H3MemoryPrice = price

			case strings.HasPrefix(skuItem.Description, "Compute optimized Instance Core"):
				pricing.C2CpuPrice = price
			case strings.HasPrefix(skuItem.Description, "Compute optimized Instance Ram"):
				pricing.C2MemoryPrice = price
			case strings.HasPrefix(skuItem.Description, "Spot Preemptible Compute optimized Instance Core"):
				pricing.SpotC2CpuPrice = price
			case strings.HasPrefix(skuItem.Description, "Spot Preemptible Compute optimized Instance Ram"):
				pricing.SpotC2MemoryPrice = price

			case strings.HasPrefix(skuItem.Description, "C2D AMD Instance Core"):
				pricing.C2DCpuPrice = price
			case strings.HasPrefix(skuItem.Description, "C2D AMD Instance Ram"):
				pricing.C2DMemoryPrice = price
			case strings.HasPrefix(skuItem.Description, "Spot Preemptible C2D AMD Instance Core"):
				pricing.SpotC2DCpuPrice = price
			case strings.HasPrefix(skuItem.Description, "Spot Preemptible C2D AMD Instance Ram"):
				pricing.SpotC2DMemoryPrice = price

			case strings.HasPrefix(skuItem.Description, "G2 Instance Core"):
				pricing.G2CpuPrice = price
			case strings.HasPrefix(skuItem.Description, "G2 Instance Ram"):
				pricing.G2MemoryPrice = price
			case strings.HasPrefix(skuItem.Description, "Spot Preemptible G2 Instance Core"):
				pricing.SpotG2DCpuPrice = price
			case strings.HasPrefix(skuItem.Description, "Spot Preemptible G2 Instance Ram"):
				pricing.SpotG2DMemoryPrice = price

			case strings.HasPrefix(skuItem.Description, "A2 Instance Core"):
				pricing.A2CpuPrice = price
			case strings.HasPrefix(skuItem.Description, "A2 Instance Ram"):
				pricing.A2MemoryPrice = price
			case strings.HasPrefix(skuItem.Description, "Spot Preemptible A2 Instance Core"):
				pricing.SpotA2CpuPrice = price
			case strings.HasPrefix(skuItem.Description, "Spot Preemptible A2 Instance Ram"):
				pricing.SpotA2MemoryPrice = price

			case strings.HasPrefix(skuItem.Description, "A3 Instance Core"):
				pricing.A3CpuPrice = price
			case strings.HasPrefix(skuItem.Description, "A3 Instance Ram"):
				pricing.A3MemoryPrice = price
			case strings.HasPrefix(skuItem.Description, "Spot Preemptible A3 Instance Core"):
				pricing.SpotA3CpuPrice = price
			case strings.HasPrefix(skuItem.Description, "Spot Preemptible A3 Instance Ram"):
				pricing.SpotA3MemoryPrice = price
			}
		}
		return nil
	})

	if err != nil {
		return GCEPriceList{}, fmt.Errorf("unable to fetch GCE cloud billing information: %w", err)
	}

	return pricing, nil
}

func GetAutopilotPricing(ctx context.Context, sku string, region string) (AutopilotPriceList, error) {
	region = NormalizeRegion(region)
	pricing := AutopilotPriceList{Region: region}

	cloudbillingService, err := cloudbilling.NewService(ctx, option.WithScopes(cloudbilling.CloudPlatformScope))
	if err != nil {
		return AutopilotPriceList{}, fmt.Errorf("unable to initialize cloud billing service: %w", err)
	}

	err = cloudbillingService.Services.Skus.List("services/"+sku).CurrencyCode("USD").Pages(ctx, func(pricingInfo *cloudbilling.ListSkusResponse) error {
		for _, skuItem := range pricingInfo.Skus {
			if !slices.Contains(skuItem.ServiceRegions, region) {
				continue
			}
			if len(skuItem.PricingInfo) == 0 || len(skuItem.PricingInfo[0].PricingExpression.TieredRates) == 0 {
				continue
			}

			rate := skuItem.PricingInfo[0].PricingExpression.TieredRates[0]
			decimal := rate.UnitPrice.Units * 1000000000
			mantissa := rate.UnitPrice.Nanos * int64(skuItem.PricingInfo[0].PricingExpression.DisplayQuantity)
			price := float64(decimal+mantissa) / 1000000000

			switch skuItem.Description {
			case "Autopilot Pod Ephemeral Storage Requests (" + region + ")":
				pricing.StoragePrice = price

			case "Autopilot Pod Memory Requests (" + region + ")":
				pricing.MemoryPrice = price

			case "Autopilot Pod mCPU Requests (" + region + ")":
				pricing.CpuPrice = price

			case "Autopilot Balanced Pod Memory Requests (" + region + ")":
				pricing.MemoryBalancedPrice = price

			case "Autopilot Balanced Pod mCPU Requests (" + region + ")":
				pricing.CpuBalancedPrice = price

			case "Autopilot Scale-Out x86 Pod Memory Requests (" + region + ")":
				pricing.MemoryScaleoutPrice = price

			case "Autopilot Scale-Out x86 Pod mCPU Requests (" + region + ")":
				pricing.CpuScaleoutPrice = price

			case "Autopilot Scale-Out Arm Pod Memory Requests (" + region + ")":
				pricing.MemoryArmScaleoutPrice = price

			case "Autopilot Scale-Out Arm Pod mCPU Requests (" + region + ")":
				pricing.CpuArmScaleoutPrice = price

			case "Autopilot Spot Pod Memory Requests (" + region + ")":
				pricing.SpotMemoryPrice = price

			case "Autopilot Spot Pod mCPU Requests (" + region + ")":
				pricing.SpotCpuPrice = price

			case "Autopilot Balanced Spot Pod Memory Requests (" + region + ")":
				pricing.SpotMemoryBalancedPrice = price

			case "Autopilot Balanced Spot Pod mCPU Requests (" + region + ")":
				pricing.SpotCpuBalancedPrice = price

			case "Autopilot Scale-Out x86 Spot Pod Memory Requests (" + region + ")":
				pricing.SpotMemoryScaleoutPrice = price

			case "Autopilot Scale-Out x86 Spot Pod mCPU Requests (" + region + ")":
				pricing.SpotCpuScaleoutPrice = price

			case "Autopilot Scale-Out Arm Spot Pod Memory Requests (" + region + ")":
				pricing.SpotArmMemoryScaleoutPrice = price

			case "Autopilot Scale-Out Arm Spot Pod mCPU Requests (" + region + ")":
				pricing.SpotArmCpuScaleoutPrice = price

			case "Autopilot NVIDIA T4 Pod mCPU Requests (" + region + ")",
				"Autopilot NVIDIA L4 Pod mCPU Requests (" + region + ")",
				"Autopilot NVIDIA A100 Pod mCPU Requests (" + region + ")",
				"Autopilot NVIDIA A100 80GB Pod mCPU Requests (" + region + ")":
				pricing.GPUPodvCPUPrice = price

			case "Autopilot NVIDIA T4 Pod Memory Requests (" + region + ")",
				"Autopilot NVIDIA L4 Pod Memory Requests (" + region + ")",
				"Autopilot NVIDIA A100 Pod Memory Requests (" + region + ")",
				"Autopilot NVIDIA A100 80GB Pod Memory Requests (" + region + ")":
				pricing.GPUPodMemoryPrice = price

			case "Autopilot NVIDIA T4 Pod GPU Requests (" + region + ")":
				pricing.NVIDIAT4PodGPUPrice = price
			case "Autopilot NVIDIA L4 Pod GPU Requests (" + region + ")":
				pricing.NVIDIAL4PodGPUPrice = price
			case "Autopilot NVIDIA A100 Pod GPU Requests (" + region + ")":
				pricing.NVIDIAA10040GPodGPUPrice = price
			case "Autopilot NVIDIA A100 80GB Pod GPU Requests (" + region + ")":
				pricing.NVIDIAA10080GPodGPUPrice = price
			case "Autopilot GPU Pod Local SSD (" + region + ")":
				pricing.GPUPodLocalSSDPrice = price

			case "Autopilot NVIDIA T4 Spot Pod mCPU Requests (" + region + ")",
				"Autopilot NVIDIA L4 Spot Pod mCPU Requests (" + region + ")",
				"Autopilot NVIDIA A100 Spot Pod mCPU Requests (" + region + ")",
				"Autopilot NVIDIA A100 80GB Spot Pod mCPU Requests (" + region + ")":
				pricing.SpotGPUPodvCPUPrice = price

			case "Autopilot NVIDIA T4 Spot Pod Memory Requests (" + region + ")",
				"Autopilot NVIDIA L4 Spot Pod Memory Requests (" + region + ")",
				"Autopilot NVIDIA A100 Spot Pod Memory Requests (" + region + ")",
				"Autopilot NVIDIA A100 80GB Spot Pod Memory Requests (" + region + ")":
				pricing.SpotGPUPodMemoryPrice = price

			case "Autopilot NVIDIA T4 Spot Pod GPU Requests (" + region + ")":
				pricing.SpotNVIDIAT4PodGPUPrice = price
			case "Autopilot NVIDIA L4 Spot Pod GPU Requests (" + region + ")":
				pricing.SpotNVIDIAL4PodGPUPrice = price
			case "Autopilot NVIDIA A100 Spot Pod GPU Requests (" + region + ")":
				pricing.SpotNVIDIAA10040GPodGPUPrice = price
			case "Autopilot NVIDIA A100 80GB Spot Pod GPU Requests (" + region + ")":
				pricing.SpotNVIDIAA10080GPodGPUPrice = price
			case "Autopilot GPU Spot Pod Local SSD (" + region + ")":
				pricing.SpotGPUPodLocalSSDPrice = price

			case "Autopilot PD Balanced Premium (" + region + ")":
				pricing.PerformancePDPricePremium = price
				pricing.SpotPerformancePDPricePremium = price
				pricing.AcceleratorPDPricePremium = price
				pricing.SpotAcceleratorPDPricePremium = price

			case "Autopilot Performance CPU Premium (" + region + ")":
				pricing.PerformanceCpuPricePremium = price
			case "Autopilot Performance Memory Premium (" + region + ")":
				pricing.PerformanceMemoryPricePremium = price
			case "Autopilot Local SSD Premium (" + region + ")":
				pricing.PerformanceLocalSSDPricePremium = price
				pricing.AcceleratorLocalSSDPricePremium = price

			case "Autopilot Spot PD Balanced Premium (" + region + ")":
				pricing.SpotPerformancePDPricePremium = price
				pricing.SpotAcceleratorPDPricePremium = price

			case "Autopilot Performance Spot CPU Premium (" + region + ")":
				pricing.SpotPerformanceCpuPricePremium = price
			case "Autopilot Performance Spot Memory Premium (" + region + ")":
				pricing.SpotPerformanceMemoryPricePremium = price
			case "Autopilot Local SSD Spot Premium (" + region + ")":
				pricing.SpotPerformanceLocalSSDPricePremium = price
				pricing.SpotAcceleratorLocalSSDPricePremium = price

			case "Autopilot Accelerator CPU Premium (" + region + ")":
				pricing.AcceleratorCpuPricePremium = price
			case "Autopilot Accelerator Memory Premium (" + region + ")":
				pricing.AcceleratorMemoryGPUPricePremium = price
			case "Autopilot T4 Premium (" + region + ")":
				pricing.AcceleratorT4GPUPricePremium = price
			case "Autopilot L4 Premium (" + region + ")":
				pricing.AcceleratorL4GPUPricePremium = price
			case "Autopilot A100 40GB Premium (" + region + ")":
				pricing.AcceleratorA10040GGPUPricePremium = price
			case "Autopilot A100 80GB Premium (" + region + ")":
				pricing.AcceleratorA10080GGPUPricePremium = price
			case "Autopilot H100 80GB Premium (" + region + ")":
				pricing.AcceleratorH100GPUPricePremium = price

			case "Autopilot Accelerator Spot CPU Premium (" + region + ")":
				pricing.SpotAcceleratorCpuPricePremium = price
			case "Autopilot Accelerator Spot Memory Premium (" + region + ")":
				pricing.SpotAcceleratorMemoryGPUPricePremium = price
			case "Autopilot T4 Spot Premium (" + region + ")":
				pricing.SpotAcceleratorT4GPUPricePremium = price
			case "Autopilot L4 Spot Premium (" + region + ")":
				pricing.SpotAcceleratorL4GPUPricePremium = price
			case "Autopilot A100 40GB Spot Premium (" + region + ")":
				pricing.SpotAcceleratorA10040GGPUPricePremium = price
			case "Autopilot A100 80GB Spot Premium (" + region + ")":
				pricing.SpotAcceleratorA10080GGPUPricePremium = price
			case "Autopilot H100 80GB Spot Premium (" + region + ")":
				pricing.SpotAcceleratorH100GPUPricePremium = price
			}
		}
		return nil
	})

	if err != nil {
		return AutopilotPriceList{}, fmt.Errorf("unable to fetch Autopilot cloud billing information: %w", err)
	}

	return pricing, nil
}

// GetMockPricing returns representative pricing rates for us-central1 (useful for testing and demo mode)
func GetMockPricing(region string) (AutopilotPriceList, GCEPriceList) {
	if region == "" {
		region = "us-central1"
	}

	ap := AutopilotPriceList{
		Region:                  region,
		StoragePrice:            0.0000706,
		CpuPrice:                0.0573,
		MemoryPrice:             0.0063421,
		CpuBalancedPrice:       0.0831,
		MemoryBalancedPrice:    0.0091933,
		CpuScaleoutPrice:       0.0722,
		MemoryScaleoutPrice:    0.0079911,
		CpuArmScaleoutPrice:    0.0515,
		MemoryArmScaleoutPrice: 0.005700,

		SpotCpuPrice:            0.0172,
		SpotMemoryPrice:         0.0019026,
		SpotCpuBalancedPrice:    0.0249,
		SpotMemoryBalancedPrice: 0.002758,
		SpotCpuScaleoutPrice:    0.0217,
		SpotMemoryScaleoutPrice: 0.0023973,
		SpotArmCpuScaleoutPrice: 0.01545,
		SpotArmMemoryScaleoutPrice: 0.00171,

		GPUPodvCPUPrice:          0.071,
		GPUPodMemoryPrice:        0.0078,
		GPUPodLocalSSDPrice:      0.0001,
		NVIDIAL4PodGPUPrice:      0.6783,
		NVIDIAT4PodGPUPrice:      0.3500,
		NVIDIAA10040GPodGPUPrice: 2.9339,
		NVIDIAA10080GPodGPUPrice: 3.6738,

		SpotGPUPodvCPUPrice:          0.0213,
		SpotGPUPodMemoryPrice:        0.00234,
		SpotGPUPodLocalSSDPrice:      0.00003,
		SpotNVIDIAL4PodGPUPrice:      0.2035,
		SpotNVIDIAT4PodGPUPrice:      0.1272,
		SpotNVIDIAA10040GPodGPUPrice: 0.8802,
		SpotNVIDIAA10080GPodGPUPrice: 1.1021,

		PerformanceCpuPricePremium:      0.0100,
		PerformanceMemoryPricePremium:   0.0011,
		PerformancePDPricePremium:       0.00005,
		PerformanceLocalSSDPricePremium: 0.00008,

		AcceleratorCpuPricePremium:        0.0120,
		AcceleratorMemoryGPUPricePremium:  0.0013,
		AcceleratorT4GPUPricePremium:      0.3500,
		AcceleratorL4GPUPricePremium:      0.6783,
		AcceleratorA10040GGPUPricePremium: 2.9339,
		AcceleratorA10080GGPUPricePremium: 3.6738,
		AcceleratorH100GPUPricePremium:    4.9500,
	}

	gce := GCEPriceList{
		Region:             region,
		H3CpuPrice:         0.045,
		H3MemoryPrice:      0.005,
		C2CpuPrice:         0.040,
		C2MemoryPrice:      0.0054,
		C2DCpuPrice:        0.038,
		C2DMemoryPrice:     0.0051,
		G2CpuPrice:         0.042,
		G2MemoryPrice:      0.0056,
		A2CpuPrice:         0.045,
		A2MemoryPrice:      0.0060,
		A3CpuPrice:         0.050,
		A3MemoryPrice:      0.0067,
		SpotC2CpuPrice:     0.012,
		SpotC2MemoryPrice:  0.0016,
		SpotC2DCpuPrice:    0.011,
		SpotC2DMemoryPrice: 0.0015,
		SpotG2DCpuPrice:    0.013,
		SpotG2DMemoryPrice: 0.0017,
		SpotA2CpuPrice:     0.014,
		SpotA2MemoryPrice:  0.0018,
		SpotA3CpuPrice:     0.015,
		SpotA3MemoryPrice:  0.0020,
	}

	return ap, gce
}
