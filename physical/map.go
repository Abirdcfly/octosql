package physical

import (
	"context"

	"github.com/cube2222/octosql"
	"github.com/cube2222/octosql/execution"
	"github.com/cube2222/octosql/physical/metadata"
	"github.com/pkg/errors"
)

type Map struct {
	Expressions []NamedExpression
	Source      Node
	Keep        bool
}

func NewMap(expressions []NamedExpression, child Node, keep bool) *Map {
	return &Map{Expressions: expressions, Source: child, Keep: keep}
}

func (node *Map) Transform(ctx context.Context, transformers *Transformers) Node {
	exprs := make([]NamedExpression, len(node.Expressions))
	for i := range node.Expressions {
		exprs[i] = node.Expressions[i].TransformNamed(ctx, transformers)
	}
	var transformed Node = &Map{
		Expressions: exprs,
		Source:      node.Source.Transform(ctx, transformers),
		Keep:        node.Keep,
	}
	if transformers.NodeT != nil {
		transformed = transformers.NodeT(transformed)
	}
	return transformed
}

func (node *Map) Materialize(ctx context.Context, matCtx *MaterializationContext) (execution.Node, error) {
	matExprs := make([]execution.NamedExpression, len(node.Expressions))
	for i := range node.Expressions {
		materialized, err := node.Expressions[i].MaterializeNamed(ctx, matCtx)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't materialize expression with index %v", i)
		}
		matExprs[i] = materialized
	}
	materialized, err := node.Source.Materialize(ctx, matCtx)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't materialize Source node")
	}

	return execution.NewMap(matExprs, materialized, node.Keep), nil
}

// This checks if this expression is just a variable with the given variable name in disguise. (for example covered by aliased expressions)
func isVariableNameRecursive(expr Expression, name octosql.VariableName) bool {
	switch expr := expr.(type) {
	case *Variable:
		return expr.Name.Equal(name)
	case *AliasedExpression:
		return isVariableNameRecursive(expr.Expr, name)
	default:
		return false
	}
}

// This gets the name of the given expression.
func getOuterName(expr Expression) octosql.VariableName {
	switch expr := expr.(type) {
	case *Variable:
		return expr.Name
	case *AliasedExpression:
		return expr.Name
	default:
		return octosql.NewVariableName("")
	}
}

func (node *Map) Metadata() *metadata.NodeMetadata {
	if node.Keep {
		return metadata.NewNodeMetadata(node.Source.Metadata().Cardinality(), node.Source.Metadata().EventTimeField())
	}

	eventTimeField := node.Source.Metadata().EventTimeField()
	var newEventTimeField octosql.VariableName
	for _, expr := range node.Expressions {
		if isVariableNameRecursive(expr, eventTimeField) {
			newEventTimeField = getOuterName(expr)
			break
		}
	}
	return metadata.NewNodeMetadata(node.Source.Metadata().Cardinality(), newEventTimeField)
}
