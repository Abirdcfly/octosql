package execution

import (
	"context"
	"fmt"

	"github.com/cube2222/octosql"
	"github.com/cube2222/octosql/docs"
	"github.com/pkg/errors"
)

type AggregatePrototype func() Aggregate

type Aggregate interface {
	docs.Documented
	AddRecord(key octosql.Value, value octosql.Value) error
	GetAggregated(key octosql.Value) (octosql.Value, error)
	String() string
}

type GroupBy struct {
	source Node
	key    []Expression

	fields              []octosql.VariableName
	aggregatePrototypes []AggregatePrototype

	as []octosql.VariableName
}

func NewGroupBy(source Node, key []Expression, fields []octosql.VariableName, aggregatePrototypes []AggregatePrototype, as []octosql.VariableName) *GroupBy {
	return &GroupBy{source: source, key: key, fields: fields, aggregatePrototypes: aggregatePrototypes, as: as}
}

func (node *GroupBy) Get(ctx context.Context, variables octosql.Variables) (RecordStream, error) {
	source, err := node.source.Get(ctx, variables)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get stream for source in group by")
	}

	aggregates := make([]Aggregate, len(node.aggregatePrototypes))
	for i := range node.aggregatePrototypes {
		aggregates[i] = node.aggregatePrototypes[i]()
	}

	return &GroupByStream{
		source:    source,
		variables: variables,

		key:    node.key,
		groups: NewHashMap(),

		fields:     node.fields,
		aggregates: aggregates,

		as: node.as,
	}, nil
}

type GroupByStream struct {
	source    RecordStream
	variables octosql.Variables

	key    []Expression
	groups *HashMap

	fields     []octosql.VariableName
	aggregates []Aggregate

	as []octosql.VariableName

	fieldNames []octosql.VariableName
	iterator   *Iterator
}

func (stream *GroupByStream) Next(ctx context.Context) (*Record, error) {
	if stream.iterator == nil {
		for {
			record, err := stream.source.Next(ctx)
			if err != nil {
				if err == ErrEndOfStream {
					stream.fieldNames = make([]octosql.VariableName, len(stream.fields))
					for i := range stream.fields {
						if len(stream.as[i]) > 0 {
							stream.fieldNames[i] = stream.as[i]
						} else {
							stream.fieldNames[i] = octosql.NewVariableName(
								fmt.Sprintf(
									"%s_%s",
									stream.fields[i].String(),
									stream.aggregates[i].String(),
								),
							)
						}
					}
					stream.iterator = stream.groups.GetIterator()
					break
				}
				return nil, errors.Wrap(err, "couldn't get next source record")
			}

			variables, err := stream.variables.MergeWith(record.AsVariables())
			if err != nil {
				return nil, errors.Wrap(err, "couldn't merge stream variables with record")
			}

			key := make([]octosql.Value, len(stream.key))
			for i := range stream.key {
				key[i], err = stream.key[i].ExpressionValue(ctx, variables)
				if err != nil {
					return nil, errors.Wrapf(err, "couldn't evaluate group key expression with index %v", i)
				}
			}

			if len(key) == 0 {
				key = append(key, octosql.MakePhantom())
			}

			err = stream.groups.Set(octosql.MakeTuple(key), octosql.Phantom{})
			if err != nil {
				return nil, errors.Wrap(err, "couldn't put group key into hashmap")
			}

			for i := range stream.aggregates {
				var value octosql.Value
				if stream.fields[i] == "*star*" {
					mapping := make(map[string]octosql.Value, len(record.Fields()))
					for _, field := range record.Fields() {
						mapping[field.Name.String()] = record.Value(field.Name)
					}
					value = octosql.MakeObject(mapping)

				} else {
					value = record.Value(stream.fields[i])
				}
				err := stream.aggregates[i].AddRecord(octosql.MakeTuple(key), value)
				if err != nil {
					return nil, errors.Wrapf(
						err,
						"couldn't add record value to aggregate %s with index %v",
						stream.aggregates[i].String(),
						i,
					)
				}
			}
		}
	}

	key, _, ok := stream.iterator.Next()
	if !ok {
		return nil, ErrEndOfStream
	}

	values := make([]octosql.Value, len(stream.aggregates))
	for i := range stream.aggregates {
		var err error
		values[i], err = stream.aggregates[i].GetAggregated(key)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't get aggregate value")
		}
	}

	return NewRecordFromSlice(stream.fieldNames, values), nil
}

func (stream *GroupByStream) Close() error {
	return stream.source.Close()
}
