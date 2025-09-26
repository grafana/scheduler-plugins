package loosebinpack

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
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
		cpuThresholdPercent    int64
		memoryThresholdPercent int64
		nodeInfo               *framework.NodeInfo
		expectedStatus         framework.Status
	}{
		{
			name:                   "node with cpu and memory utilization below threshold",
			cpuThresholdPercent:    85,
			memoryThresholdPercent: 85,
			nodeInfo:               makeNodeInfo("node1", 1000, 1000, 100, 100),
			expectedStatus:         *framework.NewStatus(framework.Success),
		},
		{
			name:                   "node with cpu and memory utilization above threshold",
			cpuThresholdPercent:    85,
			memoryThresholdPercent: 85,
			nodeInfo:               makeNodeInfo("node1", 1000, 1000, 1000, 1000),
			expectedStatus:         *framework.NewStatus(framework.Unschedulable),
		},
		{
			name:                   "node with memory utilization above threshold",
			cpuThresholdPercent:    85,
			memoryThresholdPercent: 85,
			nodeInfo:               makeNodeInfo("node1", 1000, 1000, 100, 1000),
			expectedStatus:         *framework.NewStatus(framework.Unschedulable),
		},
		{
			name:                   "node with cpu utilization above threshold",
			cpuThresholdPercent:    85,
			memoryThresholdPercent: 85,
			nodeInfo:               makeNodeInfo("node1", 1000, 1000, 1000, 100),
			expectedStatus:         *framework.NewStatus(framework.Unschedulable),
		},
		{
			name:                   "node with different thresholdes, both pass",
			cpuThresholdPercent:    50,
			memoryThresholdPercent: 75,
			nodeInfo:               makeNodeInfo("node1", 1000, 1000, 100, 100),
			expectedStatus:         *framework.NewStatus(framework.Success),
		},
		{
			name:                   "node with different thresholdes, CPU too high",
			cpuThresholdPercent:    50,
			memoryThresholdPercent: 75,
			nodeInfo:               makeNodeInfo("node1", 1000, 1000, 501, 100),
			expectedStatus:         *framework.NewStatus(framework.Unschedulable),
		},

		{
			name:                   "node with different thresholdes, Memory too high",
			cpuThresholdPercent:    50,
			memoryThresholdPercent: 75,
			nodeInfo:               makeNodeInfo("node1", 1000, 1000, 499, 751),
			expectedStatus:         *framework.NewStatus(framework.Unschedulable),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			filterPlugin, err := makePlugin(t.Context(), tc.cpuThresholdPercent, tc.memoryThresholdPercent)
			assert.NoError(t, err)
			status := filterPlugin.Filter(t.Context(), nil, nil, tc.nodeInfo)
			assert.Equal(t, tc.expectedStatus.Code(), status.Code())
		})
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
		Memory:   allocatableMemory,
	}
	n.Requested = &framework.Resource{
		MilliCPU: requestedCpu,
		Memory:   requestedMemory,
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
