package aggregates

import (
	"github.com/cube2222/octosql"
	"github.com/cube2222/octosql/docs"
	"github.com/cube2222/octosql/execution"
	"github.com/pkg/errors"
)

type First struct {
	firsts *execution.HashMap
}

func NewFirst() *First {
	return &First{
		firsts: execution.NewHashMap(),
	}
}

func (agg *First) Document() docs.Documentation {
	return docs.Section(
		agg.String(),
		docs.Body(
			docs.Section("Description", docs.Text("Takes the first received element in the group.")),
		),
	)
}

func (agg *First) AddRecord(key octosql.Value, value octosql.Value) error {
	_, previousValueExists, err := agg.firsts.Get(key)
	if err != nil {
		return errors.Wrap(err, "couldn't get current first out of hashmap")
	}

	if previousValueExists {
		return nil
	}

	err = agg.firsts.Set(key, value)
	if err != nil {
		return errors.Wrap(err, "couldn't put new first into hashmap")
	}

	return nil
}

func (agg *First) GetAggregated(key octosql.Value) (octosql.Value, error) {
	first, ok, err := agg.firsts.Get(key)
	if err != nil {
		return octosql.ZeroValue(), errors.Wrap(err, "couldn't get first out of hashmap")
	}

	if !ok {
		return octosql.ZeroValue(), errors.Errorf("first for key not found")
	}

	return first.(octosql.Value), nil
}

func (agg *First) String() string {
	return "first"
}
