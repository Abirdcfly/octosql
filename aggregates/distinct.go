package aggregates

import (
	"github.com/tidwall/btree"

	"github.com/cube2222/octosql/execution/nodes"
	"github.com/cube2222/octosql/octosql"
	"github.com/cube2222/octosql/physical"
)

func DistinctAggregateOverloads(overloads []physical.AggregateDescriptor) []physical.AggregateDescriptor {
	out := make([]physical.AggregateDescriptor, len(overloads))
	for i := range overloads {
		out[i] = physical.AggregateDescriptor{
			ArgumentType: overloads[i].ArgumentType,
			OutputType:   overloads[i].OutputType,
			TypeFn:       overloads[i].TypeFn,
			Prototype:    NewDistinctPrototype(overloads[i].Prototype),
		}
	}
	return out
}

type Distinct struct {
	items   *btree.Generic[*distinctKey]
	wrapped nodes.Aggregate
}

func NewDistinctPrototype(wrapped func() nodes.Aggregate) func() nodes.Aggregate {
	return func() nodes.Aggregate {
		return &Distinct{
			items: btree.NewGenericOptions(func(key, than *distinctKey) bool {
				return key.value.Compare(than.value) == -1
			}, btree.Options{NoLocks: true}),
			wrapped: wrapped(),
		}
	}
}

type distinctKey struct {
	value octosql.Value
	count int
}

func (c *Distinct) Add(retraction bool, value octosql.Value) bool {
	var hint btree.PathHint

	item, ok := c.items.GetHint(&distinctKey{value: value}, &hint)
	if !ok {
		item = &distinctKey{value: value, count: 0}
		c.items.SetHint(item, &hint)
	}
	if !retraction {
		item.count++
	} else {
		item.count--
	}
	if item.count == 1 && !retraction {
		c.wrapped.Add(false, value)
	} else if item.count == 0 {
		c.items.DeleteHint(item, &hint)
		c.wrapped.Add(true, value)
	}
	return c.items.Len() == 0
}

func (c *Distinct) Trigger() octosql.Value {
	return c.wrapped.Trigger()
}
