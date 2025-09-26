package loosebinpack

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type LooseBinPack struct {
	// TODO: Add fields here
}

const Name = "LooseBinPack"

var _ = framework.ScorePlugin(&LooseBinPack{})

func New(ctx context.Context, obj runtime.Object, h framework.Handle) (framework.Plugin, error) {
	return &LooseBinPack{}, nil
}

func (lbp *LooseBinPack) Name() string {
	return Name
}

func (lbp *LooseBinPack) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) (int64, *framework.Status) {
	return 0, nil
}

func (lbp *LooseBinPack) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

func (lbp *LooseBinPack) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	return nil
}
