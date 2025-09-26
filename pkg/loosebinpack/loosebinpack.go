package loosebinpack

import (
	"context"
	"errors"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"sigs.k8s.io/scheduler-plugins/apis/config"
)

type LooseBinPack struct {
	logger                 klog.Logger
	cpuThresholdPercent    float64
	memoryThresholdPercent float64
}

const Name = "LooseBinPack"

var (
	ErrNodeNotFound             = errors.New("node not found")
	ErrPodNotFound              = errors.New("pod not found")
	ErrInvalidArgs              = errors.New("invalid args")
	ErrCpuUtilizationTooHigh    = errors.New("cpu utilization too high")
	ErrMemoryUtilizationTooHigh = errors.New("memory utilization too high")
	ErrPodNoContainers          = errors.New("pod has no containers")
)

var _ = framework.FilterPlugin(&LooseBinPack{})

func New(ctx context.Context, obj runtime.Object, h framework.Handle) (framework.Plugin, error) {
	logger := klog.FromContext(ctx).WithValues("plugin", Name)
	logger.V(4).Info("Creating loosebinpack plugin")

	args, ok := obj.(*config.LooseBinPackArgs)
	if !ok {
		logger.V(4).Error(ErrInvalidArgs, "args", obj)
		return nil, ErrInvalidArgs
	}

	return &LooseBinPack{
		logger:                 logger,
		cpuThresholdPercent:    float64(args.CpuThresholdPercent) / 100,
		memoryThresholdPercent: float64(args.MemoryThresholdPercent) / 100,
	}, nil
}

func (lbp *LooseBinPack) Name() string {
	return Name
}

func (lbp *LooseBinPack) Filter(ctx context.Context, _ *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	if nodeInfo.Node() == nil {
		lbp.logger.V(4).Error(ErrNodeNotFound, ErrNodeNotFound.Error())
		return framework.NewStatus(framework.Error, ErrNodeNotFound.Error())
	}

	if pod == nil {
		lbp.logger.V(4).Error(ErrPodNotFound, ErrPodNotFound.Error())
		return framework.NewStatus(framework.Error, ErrPodNotFound.Error())
	}

	if len(pod.Spec.Containers) == 0 {
		lbp.logger.V(4).Error(ErrPodNoContainers, ErrPodNoContainers.Error())
		return framework.NewStatus(framework.Error, ErrPodNoContainers.Error())
	}

	nodeRequestedCpuAfterPod := nodeInfo.Requested.MilliCPU
	nodeRequestedMemoryAfterPod := nodeInfo.Requested.Memory

	for _, container := range pod.Spec.Containers {
		nodeRequestedCpuAfterPod += container.Resources.Requests.Cpu().ScaledValue(resource.Milli)
		nodeRequestedMemoryAfterPod += container.Resources.Requests.Memory().Value()
	}

	nodeCpuUtilization := float64(nodeRequestedCpuAfterPod) / float64(nodeInfo.Allocatable.MilliCPU)
	nodeMemoryUtilization := float64(nodeRequestedMemoryAfterPod) / float64(nodeInfo.Allocatable.Memory)

	if nodeCpuUtilization > float64(lbp.cpuThresholdPercent) {
		lbp.logger.V(4).Error(ErrCpuUtilizationTooHigh, ErrCpuUtilizationTooHigh.Error(), "node", klog.KObj(nodeInfo.Node()))
		return framework.NewStatus(framework.Unschedulable, ErrCpuUtilizationTooHigh.Error())
	}

	if nodeMemoryUtilization > float64(lbp.memoryThresholdPercent) {
		lbp.logger.V(4).Error(ErrMemoryUtilizationTooHigh, ErrMemoryUtilizationTooHigh.Error(), "node", klog.KObj(nodeInfo.Node()))
		return framework.NewStatus(framework.Unschedulable, ErrMemoryUtilizationTooHigh.Error())
	}

	return framework.NewStatus(framework.Success)
}
