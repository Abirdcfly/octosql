use std::sync::Arc;

use arrow::array::{ArrayRef, Int64Builder};
use arrow::datatypes::{DataType, Field, Schema};
use arrow::record_batch::RecordBatch;

use crate::physical::physical::*;

pub struct Map {
    source: Arc<dyn Node>,
    expressions: Vec<Arc<dyn Expression>>,
    names: Vec<Identifier>,
}

impl Map {
    pub fn new(source: Arc<dyn Node>, expressions: Vec<Arc<dyn Expression>>, names: Vec<Identifier>) -> Map {
        Map {
            source,
            expressions,
            names,
        }
    }
}

impl Node for Map {
    // TODO: Just don't allow to use retractions field as field name.
    fn schema(&self) -> Result<Arc<Schema>, Error> {
        let source_schema = self.source.schema()?;
        let mut new_schema_fields: Vec<Field> = self
            .expressions
            .iter()
            .map(|expr| {
                expr.field_meta(&vec![], &source_schema)
                    .unwrap_or_else(|err| unimplemented!())
            })
            .enumerate()
            .map(|(i, field)| Field::new(self.names[i].to_string().as_str(), field.data_type().clone(), field.is_nullable()))
            .collect();
        new_schema_fields.push(Field::new(retractions_field, DataType::Boolean, false));
        Ok(Arc::new(Schema::new(new_schema_fields)))
    }

    fn run(
        &self,
        ctx: &ExecutionContext,
        produce: ProduceFn,
        meta_send: MetaSendFn,
    ) -> Result<(), Error> {
        let output_schema = self.schema()?;

        self.source.run(
            ctx,
            &mut |produce_ctx, batch| {
                let mut new_columns: Vec<ArrayRef> = self
                    .expressions
                    .iter()
                    .map(|expr| {
                        expr.evaluate(ctx, &batch)
                            .unwrap_or_else(|err| unimplemented!())
                    })
                    .collect();
                new_columns.push(batch.column(batch.num_columns() - 1).clone());

                let new_batch = RecordBatch::try_new(output_schema.clone(), new_columns).unwrap();

                produce(produce_ctx, new_batch)?;
                Ok(())
            },
            &mut noop_meta_send,
        );
        Ok(())
    }
}

pub trait Expression: Send + Sync {
    fn field_meta(
        &self,
        context_schema: &Vec<Arc<Schema>>,
        record_schema: &Arc<Schema>,
    ) -> Result<Field, Error>;
    fn evaluate(&self, ctx: &ExecutionContext, record: &RecordBatch) -> Result<ArrayRef, Error>;
}

pub struct FieldExpression {
    field: Identifier,
}

impl FieldExpression {
    pub fn new(field: Identifier) -> FieldExpression {
        FieldExpression { field }
    }
}

// TODO: Two phases, FieldExpression and RunningFieldExpression. First gets the schema and produces the second.
impl Expression for FieldExpression {
    fn field_meta(
        &self,
        context_schema: &Vec<Arc<Schema>>,
        record_schema: &Arc<Schema>,
    ) -> Result<Field, Error> {
        Ok(record_schema
            .field_with_name(self.field.to_string().as_str())
            .unwrap()
            .clone())
    }
    fn evaluate(&self, ctx: &ExecutionContext, record: &RecordBatch) -> Result<ArrayRef, Error> {
        let record_schema: Arc<Schema> = record.schema();
        let field_index = record_schema.index_of(self.field.to_string().as_str()).unwrap();
        Ok(record.column(field_index).clone())
    }
}

pub struct Constant {
    value: ScalarValue,
}

impl Constant {
    pub fn new(value: ScalarValue) -> Constant {
        Constant { value }
    }
}

impl Expression for Constant {
    fn field_meta(
        &self,
        context_schema: &Vec<Arc<Schema>>,
        record_schema: &Arc<Schema>,
    ) -> Result<Field, Error> {
        Ok(Field::new("", self.value.data_type(), self.value == ScalarValue::Null))
    }
    fn evaluate(&self, ctx: &ExecutionContext, record: &RecordBatch) -> Result<ArrayRef, Error> {
        match self.value {
            ScalarValue::Int64(n) => {
                let mut array = Int64Builder::new(record.num_rows());
                for i in 0..record.num_rows() {
                    array.append_value(n).unwrap();
                }
                Ok(Arc::new(array.finish()) as ArrayRef)
            },
            _ => {
                dbg!(self.value.data_type());
                unimplemented!()
            },
        }
    }
}
