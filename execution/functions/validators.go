package functions

import (
	"fmt"
	"strings"

	"github.com/cube2222/octosql"
	"github.com/cube2222/octosql/docs"
	. "github.com/cube2222/octosql/execution"
)

type SingleArgumentValidator interface {
	docs.Documented
	Validate(arg octosql.Value) error
}

type all struct {
	validators []Validator
}

func All(validators ...Validator) *all {
	return &all{validators: validators}
}

func (v *all) Validate(args ...octosql.Value) error {
	for _, validator := range v.validators {
		err := validator.Validate(args...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (v *all) Document() docs.Documentation {
	childDocs := make([]docs.Documentation, len(v.validators))
	for i := range v.validators {
		childDocs[i] = v.validators[i].Document()
	}
	return docs.List(childDocs...)
}

type singleAll struct {
	validators []SingleArgumentValidator
}

func SingleAll(validators ...SingleArgumentValidator) *singleAll {
	return &singleAll{validators: validators}
}

func (v *singleAll) Validate(args octosql.Value) error {
	for _, validator := range v.validators {
		err := validator.Validate(args)
		if err != nil {
			return err
		}
	}
	return nil
}

func (v *singleAll) Document() docs.Documentation {
	childDocs := make([]docs.Documentation, len(v.validators))
	for i := range v.validators {
		childDocs[i] = v.validators[i].Document()
	}
	return docs.List(childDocs...)
}

type oneOf struct {
	validators []Validator
}

func OneOf(validators ...Validator) *oneOf {
	return &oneOf{validators: validators}
}

func (v *oneOf) Validate(args ...octosql.Value) error {
	errs := make([]error, len(v.validators))
	for i, validator := range v.validators {
		errs[i] = validator.Validate(args...)
		if errs[i] == nil {
			return nil
		}
	}

	return fmt.Errorf("none of the conditions have been met: %+v", errs)
}

func (v *oneOf) Document() docs.Documentation {
	childDocs := make([]docs.Documentation, len(v.validators))
	for i := range v.validators {
		childDocs[i] = v.validators[i].Document()
	}

	return docs.Paragraph(docs.Text("must satisfy one of"), docs.List(childDocs...))
}

type singleOneOf struct {
	validators []SingleArgumentValidator
}

func SingleOneOf(validators ...SingleArgumentValidator) *singleOneOf {
	return &singleOneOf{validators: validators}
}

func (v *singleOneOf) Validate(arg octosql.Value) error {
	errs := make([]error, len(v.validators))
	for i, validator := range v.validators {
		errs[i] = validator.Validate(arg)
		if errs[i] == nil {
			return nil
		}
	}

	return fmt.Errorf("none of the conditions have been met: %+v", errs)
}

func (v *singleOneOf) Document() docs.Documentation {
	childDocs := make([]docs.Documentation, len(v.validators))
	for i := range v.validators {
		childDocs[i] = v.validators[i].Document()
	}

	return docs.Paragraph(docs.Text("must satisfy one of the following"), docs.List(childDocs...))
}

type ifArgPresent struct {
	i         int
	validator Validator
}

func IfArgPresent(i int, validator Validator) *ifArgPresent {
	return &ifArgPresent{i: i, validator: validator}
}

func (v *ifArgPresent) Validate(args ...octosql.Value) error {
	if len(args) < v.i+1 {
		return nil
	}
	return v.validator.Validate(args...)
}

func (v *ifArgPresent) Document() docs.Documentation {
	return docs.Paragraph(
		docs.Text(fmt.Sprintf("if the %s argument is provided, then", docs.Ordinal(v.i+1))),
		v.validator.Document(),
	)
}

type atLeastNArgs struct {
	n int
}

func AtLeastNArgs(n int) *atLeastNArgs {
	return &atLeastNArgs{n: n}
}

func (v *atLeastNArgs) Validate(args ...octosql.Value) error {
	if len(args) < v.n {
		return fmt.Errorf("expected at least %s, but got %v", argumentCount(v.n), len(args))
	}
	return nil
}

func (v *atLeastNArgs) Document() docs.Documentation {
	return docs.Text(fmt.Sprintf("at least %s may be provided", argumentCount(v.n)))
}

type atMostNArgs struct {
	n int
}

func AtMostNArgs(n int) *atMostNArgs {
	return &atMostNArgs{n: n}
}

func (v *atMostNArgs) Validate(args ...octosql.Value) error {
	if len(args) > v.n {
		return fmt.Errorf("expected at most %s, but got %v", argumentCount(v.n), len(args))
	}
	return nil
}

func (v *atMostNArgs) Document() docs.Documentation {
	return docs.Text(fmt.Sprintf("at most %s may be provided", argumentCount(v.n)))
}

type exactlyNArgs struct {
	n int
}

func ExactlyNArgs(n int) *exactlyNArgs {
	return &exactlyNArgs{n: n}
}

func (v *exactlyNArgs) Validate(args ...octosql.Value) error {
	if len(args) != v.n {
		return fmt.Errorf("expected exactly %s, but got %v", argumentCount(v.n), len(args))
	}
	return nil
}

func (v *exactlyNArgs) Document() docs.Documentation {
	return docs.Text(fmt.Sprintf("exactly %s must be provided", argumentCount(v.n)))
}

type typeOf struct {
	wantedType octosql.Value
}

func TypeOf(wantedType octosql.Value) *typeOf {
	return &typeOf{wantedType: wantedType}
}

func (v *typeOf) Validate(arg octosql.Value) error {
	if v.wantedType.GetType() != arg.GetType() {
		return fmt.Errorf("expected type %v but got %v", v.wantedType.GetType(), arg.GetType())
	}
	return nil
}

func (v *typeOf) Document() docs.Documentation {
	return docs.Paragraph(docs.Text("must be of type"), v.wantedType.Document())
}

type valueOf struct {
	values []octosql.Value
}

func ValueOf(values ...octosql.Value) *valueOf {
	return &valueOf{values: values}
}

func (v *valueOf) Validate(arg octosql.Value) error {
	for i := range v.values {
		if octosql.AreEqual(v.values[i], arg) {
			return nil
		}
	}

	values := make([]string, len(v.values))
	for i := range v.values {
		values[i] = v.values[i].String()
	}

	return fmt.Errorf(
		"argument must be one of: [%s], got %v",
		strings.Join(values, ", "),
		arg,
	)
}

func (v *valueOf) Document() docs.Documentation {
	values := make([]string, len(v.values))
	for i := range v.values {
		values[i] = v.values[i].String()
	}
	return docs.Paragraph(
		docs.Text(
			fmt.Sprintf(
				"must be one of: [%s]",
				strings.Join(values, ", "),
			),
		),
	)
}

type arg struct {
	i         int
	validator SingleArgumentValidator
}

func Arg(i int, validator SingleArgumentValidator) *arg {
	return &arg{i: i, validator: validator}
}

func (v *arg) Validate(args ...octosql.Value) error {
	if err := v.validator.Validate(args[v.i]); err != nil {
		return fmt.Errorf("bad argument at index %v: %v", v.i, err)
	}
	return nil
}

func (v *arg) Document() docs.Documentation {
	return docs.Paragraph(
		docs.Text(fmt.Sprintf("the %s argument", docs.Ordinal(v.i+1))),
		v.validator.Document(),
	)
}

type allArgs struct {
	validator SingleArgumentValidator
}

func AllArgs(validator SingleArgumentValidator) *allArgs {
	return &allArgs{validator: validator}
}

func (v *allArgs) Validate(args ...octosql.Value) error {
	for i := range args {
		if err := v.validator.Validate(args[i]); err != nil {
			return fmt.Errorf("bad argument at index %v: %v", i, err)
		}
	}
	return nil
}

func (v *allArgs) Document() docs.Documentation {
	return docs.Paragraph(
		docs.Text("all arguments"),
		v.validator.Document(),
	)
}

func argumentCount(n int) string {
	switch n {
	case 1:
		return "1 argument"
	default:
		return fmt.Sprintf("%d arguments", n)
	}
}
