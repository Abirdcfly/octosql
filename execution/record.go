package execution

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/mitchellh/hashstructure"
	"github.com/pkg/errors"

	"github.com/cube2222/octosql"
)

type Field struct {
	Name octosql.VariableName
}

func NewID(id string) ID {
	return ID{
		ID: id,
	}
}

func (id ID) Show() string {
	return id.ID
}

type RecordOption func(stream *Record)

func WithUndo() RecordOption {
	return func(r *Record) {
		r.Metadata.Undo = true
	}
}

func WithEventTimeField(field octosql.VariableName) RecordOption {
	return func(r *Record) {
		r.Metadata.EventTimeField = field.String()
	}
}

func WithMetadataFrom(base *Record) RecordOption {
	return func(r *Record) {
		r.Metadata = base.Metadata
	}
}

func WithID(id ID) RecordOption {
	return func(rec *Record) {
		rec.Metadata.Id = &id
	}
}

func NewRecord(fields []octosql.VariableName, data map[octosql.VariableName]octosql.Value, opts ...RecordOption) *Record {
	dataInner := make([]octosql.Value, len(fields))
	for i := range fields {
		dataInner[i] = data[fields[i]]
	}
	return NewRecordFromSlice(fields, dataInner, opts...)
}

func NewRecordFromSlice(fields []octosql.VariableName, data []octosql.Value, opts ...RecordOption) *Record {
	stringFields := octosql.VariableNamesToStrings(fields)

	pointerData := octosql.GetPointersFromValues(data)

	r := &Record{
		FieldNames: stringFields,
		Data:       pointerData,
		Metadata:   &Metadata{},
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func (r *Record) Value(field octosql.VariableName) octosql.Value {
	if field.Source() == "sys" {
		switch field.Name() {
		case "undo":
			return octosql.MakeBool(r.IsUndo())
		case "event_time_field":
			eventTimeField := r.EventTimeField()
			return octosql.MakeString(eventTimeField.String())
		default:
			return octosql.MakeNull()
		}
	}

	stringField := field.String()

	for i := range r.FieldNames {
		if r.FieldNames[i] == stringField {
			return *r.Data[i]
		}
	}
	return octosql.MakeNull()
}

func (r *Record) Fields() []Field {
	fields := make([]Field, 0)
	for _, fieldName := range r.FieldNames {
		fields = append(fields, Field{
			Name: octosql.NewVariableName(fieldName),
		})
	}

	if len(r.EventTimeField()) > 0 {
		fields = append(fields, Field{
			Name: octosql.NewVariableName("sys.event_time_field"),
		})
	}
	if r.IsUndo() {
		fields = append(fields, Field{
			Name: octosql.NewVariableName("sys.undo"),
		})
	}

	return fields
}

func (r *Record) AsVariables() octosql.Variables {
	out := make(octosql.Variables)
	for i := range r.FieldNames {
		out[octosql.NewVariableName(r.FieldNames[i])] = *r.Data[i]
	}

	return out
}

func (r *Record) AsTuple() octosql.Value {
	return octosql.MakeTuple(octosql.GetValuesFromPointers(r.Data))
}

func (r *Record) Equal(other *Record) bool {
	return proto.Equal(r, other)
}

func (r *Record) Show() string {
	parts := make([]string, len(r.FieldNames))
	for i := range r.FieldNames {
		parts[i] = fmt.Sprintf("%s: %s", r.FieldNames[i], r.Data[i].Show())
	}

	return fmt.Sprintf("{%s}", strings.Join(parts, ", "))
}

func (r *Record) IsUndo() bool {
	if r.Metadata != nil {
		return r.Metadata.Undo
	}

	return false
}

func (r *Record) EventTime() octosql.Value {
	eventVarName := r.EventTimeField()
	return r.Value(eventVarName)
}

func (r *Record) ID() ID {
	if r.Metadata != nil {
		return *r.Metadata.Id
	}

	return ID{}
}

func (r *Record) EventTimeField() octosql.VariableName {
	if r.Metadata != nil {
		return octosql.NewVariableName(r.Metadata.EventTimeField)
	}

	return octosql.NewVariableName("")
}

func (r *Record) GetVariableNames() []octosql.VariableName {
	return octosql.StringsToVariableNames(r.FieldNames)
}

func (r *Record) Hash() (uint64, error) {
	values := octosql.GetValuesFromPointers(r.Data)
	fields := make([]octosql.Value, len(r.FieldNames))

	for i, name := range r.FieldNames {
		fields[i] = octosql.MakeString(name)
	}

	return hashstructure.Hash(append(values, fields...), nil)
}

type RecordStream interface {
	Next(ctx context.Context) (*Record, error)
	io.Closer
}

var ErrEndOfStream = errors.New("end of stream")

var ErrNotFound = errors.New("not found")
