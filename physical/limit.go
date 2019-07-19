package physical

import (
	"context"

	"github.com/cube2222/octosql/execution"
	"github.com/pkg/errors"
)

type Limit struct {
	data      Node
	limitExpr Expression
}

func NewLimit(data Node, expr Expression) *Limit {
	return &Limit{data: data, limitExpr: expr}
}

func (node *Limit) Transform(ctx context.Context, transformers *Transformers) Node {
	var transformed Node = &Limit{
		data:      node.data.Transform(ctx, transformers),
		limitExpr: node.limitExpr.Transform(ctx, transformers),
	}
	if transformers.NodeT != nil {
		transformed = transformers.NodeT(transformed)
	}
	return transformed
}

func (node *Limit) Materialize(ctx context.Context, matCtx *MaterializationContext) (execution.Node, error) {
	dataNode, err := node.data.Materialize(ctx, matCtx)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't materialize data node")
	}

	limitExpr, err := node.limitExpr.Materialize(ctx, matCtx)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't materialize limit expression")
	}

	return execution.NewLimit(dataNode, limitExpr), nil
}
