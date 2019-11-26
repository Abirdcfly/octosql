package execution

import (
	"github.com/cube2222/octosql"

	"context"

	"github.com/pkg/errors"
)

type Filter struct {
	formula Formula
	source  Node
}

func NewFilter(formula Formula, child Node) *Filter {
	return &Filter{formula: formula, source: child}
}

func (node *Filter) Get(ctx context.Context, variables octosql.Variables) (RecordStream, error) {
	recordStream, err := node.source.Get(ctx, variables)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get record stream")
	}

	return &FilteredStream{
		formula:   node.formula,
		variables: variables,
		source:    recordStream,
	}, nil
}

type FilteredStream struct {
	formula   Formula
	variables octosql.Variables
	source    RecordStream
}

func (stream *FilteredStream) Close() error {
	err := stream.source.Close()
	if err != nil {
		return errors.Wrap(err, "Couldn't close underlying stream")
	}

	return nil
}

func (stream *FilteredStream) Next(ctx context.Context) (*Record, error) {
	for {
		record, err := stream.source.Next(ctx)
		if err != nil {
			if err == ErrEndOfStream {
				return nil, ErrEndOfStream
			}
			return nil, errors.Wrap(err, "couldn't get source record")
		}

		variables, err := stream.variables.MergeWith(record.AsVariables())
		if err != nil {
			return nil, errors.Wrap(err, "couldn't merge given variables with record variables")
		}

		predicate, err := stream.formula.Evaluate(ctx, variables)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't evaluate formula")
		}

		if predicate {
			return record, nil
		}
	}
}
