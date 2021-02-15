package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"

	"github.com/cube2222/octosql/execution"
	"github.com/cube2222/octosql/octosql"
	"github.com/cube2222/octosql/optimizer"
	"github.com/cube2222/octosql/physical"
)

type impl struct {
	config *Config
	schema physical.Schema
	table  string
}

func (impl *impl) Schema() (physical.Schema, error) {
	return impl.schema, nil
}

func (impl *impl) Materialize(ctx context.Context, env physical.Environment, pushedDownPredicates []physical.Expression) (execution.Node, error) {
	// Prepare statement
	db, err := connect(impl.config)
	if err != nil {
		return nil, fmt.Errorf("couldn't connect to database: %w", err)
	}
	fields := make([]string, len(impl.schema.Fields))
	for index := range impl.schema.Fields {
		fields[index] = impl.schema.Fields[index].Name
	}

	predicateSQL, placeholderExpressions := predicatesToSQL(pushedDownPredicates)
	stmt, err := db.PrepareEx(ctx, uuid.Must(uuid.NewV4()).String(), fmt.Sprintf("SELECT %s FROM %s WHERE %s", strings.Join(fields, ", "), impl.table, predicateSQL), nil)
	if err != nil {
		return nil, fmt.Errorf("couldn't prepare statement: %w", err)
	}

	executionPlaceholderExprs := make([]execution.Expression, len(placeholderExpressions))
	for i := range placeholderExpressions {
		expr, err := placeholderExpressions[i].Materialize(ctx, env)
		if err != nil {
			return nil, fmt.Errorf("couldn't materialize pushed-down predicate placeholder expression: %w", err)
		}
		executionPlaceholderExprs[i] = expr
	}

	return &DatasourceExecuting{
		fields:           impl.schema.Fields,
		table:            impl.table,
		placeholderExprs: executionPlaceholderExprs,
		db:               db,
		stmt:             stmt,
	}, nil
}

func predicatesToSQL(predicates []physical.Expression) (predicateSQL string, placeholderExprs []physical.Expression) {
	if len(predicates) == 0 {
		return "(TRUE)", nil
	}

	var builder strings.Builder
	var placeholderExpressions []physical.Expression
	var predicateExpr physical.Expression
	if len(predicates) == 1 {
		predicateExpr = predicates[0]
	} else {
		predicateExpr = physical.Expression{
			Type:           octosql.Boolean,
			ExpressionType: physical.ExpressionTypeAnd,
			And: &physical.And{
				Arguments: predicates,
			},
		}
	}

	predicateToSQL(&builder, &placeholderExpressions, predicateExpr)
	return builder.String(), placeholderExpressions
}

func predicateToSQL(builder *strings.Builder, placeholderExpressions *[]physical.Expression, expression physical.Expression) {
	builder.WriteString(" (")
	switch expression.ExpressionType {
	case physical.ExpressionTypeVariable:
		if expression.Variable.IsLevel0 {
			builder.WriteString(expression.Variable.Name)
		} else {
			builder.WriteString(fmt.Sprintf("$%d", len(*placeholderExpressions)+1))
			*placeholderExpressions = append(*placeholderExpressions, expression)
		}
	case physical.ExpressionTypeConstant:
		switch expression.Type.TypeID {
		case octosql.TypeIDNull:
			builder.WriteString("NULL")
		case octosql.TypeIDInt, octosql.TypeIDFloat, octosql.TypeIDBoolean,
			octosql.TypeIDString, octosql.TypeIDTime:
			builder.WriteString(fmt.Sprintf("$%d", len(*placeholderExpressions)+1))
			*placeholderExpressions = append(*placeholderExpressions, expression)
		default:
			panic("invalid pushed down predicate constant")
		}
	case physical.ExpressionTypeFunctionCall:
		switch expression.FunctionCall.Name {
		case ">", ">=", "=", "<=", "<": // Operators
			predicateToSQL(builder, placeholderExpressions, expression.FunctionCall.Arguments[0])
			builder.WriteString(expression.FunctionCall.Name)
			predicateToSQL(builder, placeholderExpressions, expression.FunctionCall.Arguments[1])
		default:
			panic("invalid pushed down predicate function")
		}
	case physical.ExpressionTypeAnd:
		for i := range expression.And.Arguments {
			predicateToSQL(builder, placeholderExpressions, expression.And.Arguments[i])
			if i != len(expression.And.Arguments)-1 {
				builder.WriteString("AND")
			}
		}
	case physical.ExpressionTypeOr:
		for i := range expression.And.Arguments {
			predicateToSQL(builder, placeholderExpressions, expression.And.Arguments[i])
			if i != len(expression.And.Arguments)-1 {
				builder.WriteString(" OR ")
			}
		}
	default:
		panic("invalid pushed down predicate")
	}
	builder.WriteString(") ")
}

func (impl *impl) PushDownPredicates(newPredicates, pushedDownPredicates []physical.Expression) (rejected []physical.Expression, newPushedDown []physical.Expression, changed bool) {
	newPushedDown = make([]physical.Expression, len(pushedDownPredicates))
	copy(newPushedDown, pushedDownPredicates)
	for _, pred := range newPredicates {
		isOk := true
		predicateChecker := optimizer.Transformers{
			ExpressionTransformer: func(expr physical.Expression) physical.Expression {
				switch expr.ExpressionType {
				case physical.ExpressionTypeVariable:
				case physical.ExpressionTypeConstant:
					switch expr.Type.TypeID {
					case octosql.TypeIDInt, octosql.TypeIDFloat, octosql.TypeIDBoolean,
						octosql.TypeIDString, octosql.TypeIDTime, octosql.TypeIDNull:
					default:
						isOk = false
					}
				case physical.ExpressionTypeFunctionCall:
					switch expr.FunctionCall.Name {
					case ">", ">=", "=", "<", "<=":
					default:
						isOk = false
					}
				case physical.ExpressionTypeAnd:
				case physical.ExpressionTypeOr:
				default:
					isOk = false
				}
				return expr
			},
		}
		predicateChecker.TransformExpr(pred)
		if isOk {
			newPushedDown = append(newPushedDown, pred)
		} else {
			rejected = append(rejected, pred)
		}
	}
	changed = len(newPushedDown) > len(pushedDownPredicates)
	return
}