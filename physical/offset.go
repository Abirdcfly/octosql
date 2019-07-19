package physical

import (
	"context"

	"github.com/cube2222/octosql/execution"
	"github.com/pkg/errors"
)

type Offset struct {
	data       Node
	offsetExpr Expression
}

func NewOffset(data Node, expr Expression) *Offset {
	return &Offset{data: data, offsetExpr: expr}
}

func (node *Offset) Transform(ctx context.Context, transformers *Transformers) Node {
	var transformed Node = &Offset{
		data:       node.data.Transform(ctx, transformers),
		offsetExpr: node.offsetExpr.Transform(ctx, transformers),
	}
	if transformers.NodeT != nil {
		transformed = transformers.NodeT(transformed)
	}
	return transformed
}

func (node *Offset) Materialize(ctx context.Context, matCtx *MaterializationContext) (execution.Node, error) {
	dataNode, err := node.data.Materialize(ctx, matCtx)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't materialize data node")
	}

	offsetExpr, err := node.offsetExpr.Materialize(ctx, matCtx)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't materialize offset expression")
	}

	return execution.NewOffset(dataNode, offsetExpr), nil
}
