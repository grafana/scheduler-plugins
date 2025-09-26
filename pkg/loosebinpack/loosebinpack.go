package loosebinpack

import (
	"context"
	"errors"

	v1 "k8s.io/api/core/v1"
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
	ErrInvalidArgs              = errors.New("invalid args")
	ErrCpuUtilizationTooHigh    = errors.New("cpu utilization too high")
	ErrMemoryUtilizationTooHigh = errors.New("memory utilization too high")
)

var _ = framework.FilterPlugin(&LooseBinPack{})

func New(ctx context.Context, obj runtime.Object, h framework.Handle) (framework.Plugin, error) {
	logger := klog.FromContext(ctx).WithValues("plugin", Name)
	logger.V(4).Info("Creating new instance of the NetworkOverhead plugin")

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

func (lbp *LooseBinPack) Filter(ctx context.Context, _ *framework.CycleState, _ *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	if nodeInfo.Node() == nil {
		lbp.logger.V(4).Error(ErrNodeNotFound, ErrNodeNotFound.Error())
		return framework.NewStatus(framework.Error, ErrNodeNotFound.Error())
	}

	nodeAllocatableCpu := nodeInfo.Allocatable.MilliCPU
	nodeAllocatableMemory := nodeInfo.Allocatable.Memory

	nodeRequestedCpu := nodeInfo.Requested.MilliCPU
	nodeRequestedMemory := nodeInfo.Requested.Memory

	nodeCpuUtilization := float64(nodeRequestedCpu) / float64(nodeAllocatableCpu)
	nodeMemoryUtilization := float64(nodeRequestedMemory) / float64(nodeAllocatableMemory)

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
