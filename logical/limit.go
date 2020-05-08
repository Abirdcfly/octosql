package logical

import (
	"context"

	"github.com/cube2222/octosql"
	"github.com/cube2222/octosql/graph"
	"github.com/cube2222/octosql/physical"
	"github.com/pkg/errors"
)

type Limit struct {
	data      Node
	limitExpr Expression
}

func NewLimit(data Node, expr Expression) Node {
	return &Limit{data: data, limitExpr: expr}
}

func (node *Limit) Physical(ctx context.Context, physicalCreator *PhysicalPlanCreator) ([]physical.Node, octosql.Variables, error) {
	sourceNodes, variables, err := node.data.Physical(ctx, physicalCreator)
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't get physical plan for source nodes")
	}

	limitExpr, limitVariables, err := node.limitExpr.Physical(ctx, physicalCreator)
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't get physical plan for limit expression")
	}
	variables, err = variables.MergeWith(limitVariables)
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't get limit node variables")
	}

	// Limit operates on a single, joined stream.
	outNodes := physical.NewShuffle(1, physical.NewConstantStrategy(0), sourceNodes)

	return []physical.Node{physical.NewLimit(outNodes[0], limitExpr)}, variables, nil
}

func (node *Limit) Visualize() *graph.Node {
	n := graph.NewNode("Limit")
	if node.limitExpr != nil {
		n.AddChild("limit", node.limitExpr.Visualize())
	}
	if node.data != nil {
		n.AddChild("data", node.data.Visualize())
	}
	return n
}
