package execution

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/cube2222/octosql/octosql"
)

type Node interface {
	Run(ctx ExecutionContext, produce ProduceFn, metaSend MetaSendFn) error
}

type ExecutionContext struct {
	context.Context
	VariableContext *VariableContext
}

func (ctx ExecutionContext) WithRecord(record Record) ExecutionContext {
	return ExecutionContext{
		Context:         ctx.Context,
		VariableContext: ctx.VariableContext.WithRecord(record),
	}
}

type VariableContext struct {
	Parent    *VariableContext
	Values    []octosql.Value
	EventTime time.Time
}

func (varCtx *VariableContext) WithRecord(record Record) *VariableContext {
	return &VariableContext{
		Parent:    varCtx,
		Values:    record.Values,
		EventTime: record.EventTime,
	}
}

type ProduceFn func(ctx ProduceContext, record Record) error

type ProduceContext struct {
	context.Context
}

func ProduceFromExecutionContext(ctx ExecutionContext) ProduceContext {
	return ProduceContext{
		Context: ctx.Context,
	}
}

type Record struct {
	Values     []octosql.Value
	Retraction bool
	EventTime  time.Time
}

// Functional options?
func NewRecord(values []octosql.Value, retraction bool, eventTime time.Time) Record {
	return Record{
		Values:     values,
		Retraction: retraction,
		EventTime:  eventTime,
	}
}

func (record Record) String() string {
	builder := strings.Builder{}
	builder.WriteString("{")
	if !record.Retraction {
		builder.WriteString("+")
	} else {
		builder.WriteString("-")
	}
	builder.WriteString("| ")
	for i := range record.Values {
		builder.WriteString(record.Values[i].String())
		if i != len(record.Values)-1 {
			builder.WriteString(", ")
		}
	}
	builder.WriteString(" |}")
	return builder.String()
}

type MetaSendFn func(ctx ProduceContext, msg MetadataMessage) error

type MetadataMessage struct {
	Type      MetadataMessageType
	Watermark time.Time
}

type MetadataMessageType int

const (
	MetadataMessageTypeWatermark MetadataMessageType = iota
)

var WatermarkMaxValue = (&time.Time{}).Add(time.Duration(math.MaxInt64))
