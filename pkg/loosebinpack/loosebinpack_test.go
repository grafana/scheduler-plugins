package loosebinpack

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"sigs.k8s.io/scheduler-plugins/apis/config"
)

func TestNew(t *testing.T) {
	pluginArgs := &config.LooseBinPackArgs{
		CpuThresholdPercent:    85,
		MemoryThresholdPercent: 85,
	}
	_, err := New(t.Context(), pluginArgs, nil)

	assert.NoError(t, err)
}

func TestNodeFilter(t *testing.T) {
	tt := []struct {
		name                   string
		pod                    *v1.Pod
		cpuThresholdPercent    int64
		memoryThresholdPercent int64
		nodeInfo               *framework.NodeInfo
		expectedStatus         framework.Status
	}{
		{
			name:                   "pod with no containers",
			pod:                    makePod("pod1"),
			cpuThresholdPercent:    85,
			memoryThresholdPercent: 85,
			nodeInfo:               makeNodeInfo("node1", 1000, 1000, 100, 100),
			expectedStatus:         *framework.NewStatus(framework.Error, ErrPodNoContainers.Error()),
		},
		{
			name:                   "node with cpu and memory utilization below threshold",
			pod:                    makePod("pod1", containerReq{CpuReq: 100, MemoryReq: 100}),
			cpuThresholdPercent:    85,
			memoryThresholdPercent: 85,
			nodeInfo:               makeNodeInfo("node1", 1000, 1000, 100, 100),
			expectedStatus:         *framework.NewStatus(framework.Success),
		},
		{
			name:                   "node with cpu and memory utilization above threshold",
			pod:                    makePod("pod1", containerReq{CpuReq: 100, MemoryReq: 100}),
			cpuThresholdPercent:    85,
			memoryThresholdPercent: 85,
			nodeInfo:               makeNodeInfo("node1", 1000, 1000, 900, 900),
			expectedStatus:         *framework.NewStatus(framework.Unschedulable, ErrCpuUtilizationTooHigh.Error()),
		},
		{
			name:                   "node with cpu and memory utilization below threshold until pod is added",
			pod:                    makePod("pod1", containerReq{CpuReq: 100, MemoryReq: 100}),
			cpuThresholdPercent:    85,
			memoryThresholdPercent: 85,
			nodeInfo:               makeNodeInfo("node1", 1000, 1000, 800, 800),
			expectedStatus:         *framework.NewStatus(framework.Unschedulable, ErrCpuUtilizationTooHigh.Error()),
		},
		{
			name:                   "node with cpu and memory utilization below threshold until pod is added - two containers. first would not cause failure, second will",
			pod:                    makePod("pod1", containerReq{CpuReq: 100, MemoryReq: 100}, containerReq{CpuReq: 100, MemoryReq: 100}),
			cpuThresholdPercent:    85,
			memoryThresholdPercent: 85,
			nodeInfo:               makeNodeInfo("node1", 1000, 1000, 700, 700),
			expectedStatus:         *framework.NewStatus(framework.Unschedulable, ErrCpuUtilizationTooHigh.Error()),
		},
		{
			name:                   "node with cpu and memory utilization below threshold - two containers. first would not cause failure, second wont either",
			pod:                    makePod("pod1", containerReq{CpuReq: 100, MemoryReq: 100}, containerReq{CpuReq: 100, MemoryReq: 100}),
			cpuThresholdPercent:    85,
			memoryThresholdPercent: 85,
			nodeInfo:               makeNodeInfo("node1", 1000, 1000, 600, 600),
			expectedStatus:         *framework.NewStatus(framework.Success),
		},
		{
			name:                   "node with cpu and memory utilization below threshold - two containers. first would not cause failure, second will cause CPU threshold failure",
			pod:                    makePod("pod1", containerReq{CpuReq: 100, MemoryReq: 100}, containerReq{CpuReq: 151, MemoryReq: 100}),
			cpuThresholdPercent:    85,
			memoryThresholdPercent: 85,
			nodeInfo:               makeNodeInfo("node1", 1000, 1000, 600, 600),
			expectedStatus:         *framework.NewStatus(framework.Unschedulable, ErrCpuUtilizationTooHigh.Error()),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			filterPlugin, err := makePlugin(t.Context(), tc.cpuThresholdPercent, tc.memoryThresholdPercent)
			assert.NoError(t, err)
			status := filterPlugin.Filter(t.Context(), nil, tc.pod, tc.nodeInfo)
			assert.Equal(t, tc.expectedStatus.Code(), status.Code())
			assert.Equal(t, tc.expectedStatus.Message(), status.Message())
		})
	}
}

type containerReq struct {
	CpuReq    int64
	MemoryReq int64
}

func makeContainer(name string, cpuReq, memoryReq int64) v1.Container {
	return v1.Container{
		Name: name,
		Resources: v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse(strconv.FormatInt(cpuReq, 10) + "m"),
				v1.ResourceMemory: resource.MustParse(strconv.FormatInt(memoryReq, 10) + "Ki"),
			},
		},
	}
}

func makePod(name string, containerReq ...containerReq) *v1.Pod {
	containers := make([]v1.Container, len(containerReq))
	for i, req := range containerReq {
		containers[i] = makeContainer(fmt.Sprintf("%s-%d", name, i), req.CpuReq, req.MemoryReq)
	}

	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.PodSpec{
			Containers: containers,
		},
	}
}

func makeNodeInfo(name string, allocatableCpu, allocatableMemory, requestedCpu, requestedMemory int64) *framework.NodeInfo {
	n := framework.NewNodeInfo()
	n.SetNode(&v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	})
	n.Allocatable = &framework.Resource{
		MilliCPU: allocatableCpu,
		Memory:   allocatableMemory * 1024,
	}
	n.Requested = &framework.Resource{
		MilliCPU: requestedCpu,
		Memory:   requestedMemory * 1024,
	}
	return n
}

func makePlugin(ctx context.Context, cpuThresholdPercent, memoryThresholdPercent int64) (framework.FilterPlugin, error) {
	pluginArgs := &config.LooseBinPackArgs{
		CpuThresholdPercent:    cpuThresholdPercent,
		MemoryThresholdPercent: memoryThresholdPercent,
	}
	plugin, err := New(ctx, pluginArgs, nil)
	if err != nil {
		return nil, err
	}
	filterPlugin, ok := plugin.(framework.FilterPlugin)
	if !ok {
		return nil, fmt.Errorf("plugin is not a filter plugin")
	}

	return filterPlugin, nil
}
