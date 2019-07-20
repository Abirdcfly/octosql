package optimizer

import (
	"context"
	"reflect"
	"testing"

	"github.com/cube2222/octosql"
	"github.com/cube2222/octosql/physical"
)

func TestMergeRequalifiers(t *testing.T) {
	type args struct {
		plan physical.Node
	}
	tests := []struct {
		name string
		args args
		want physical.Node
	}{
		{
			name: "simple single merge",
			args: args{
				plan: &physical.Requalifier{
					Qualifier: "a",
					Source: &physical.Requalifier{
						Qualifier: "b",
						Source: &PlaceholderNode{
							Name: "stub",
						},
					},
				},
			},
			want: &physical.Requalifier{
				Qualifier: "a",
				Source: &PlaceholderNode{
					Name: "stub",
				},
			},
		},
		{
			name: "multi merge",
			args: args{
				plan: &physical.Requalifier{
					Qualifier: "a",
					Source: &physical.Requalifier{
						Qualifier: "b",
						Source: &physical.Requalifier{
							Qualifier: "c",
							Source: &physical.Requalifier{
								Qualifier: "d",
								Source: &physical.Requalifier{
									Qualifier: "e",
									Source: &physical.Requalifier{
										Qualifier: "f",
										Source: &physical.Requalifier{
											Qualifier: "g",
											Source: &physical.Requalifier{
												Qualifier: "h",
												Source: &PlaceholderNode{
													Name: "stub",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: &physical.Requalifier{
				Qualifier: "a",
				Source: &PlaceholderNode{
					Name: "stub",
				},
			},
		},
		{
			name: "deeper merge",
			args: args{
				plan: &physical.Map{
					Expressions: []physical.NamedExpression{physical.NewVariable("expr")},
					Source: &physical.Requalifier{
						Qualifier: "a",
						Source: &physical.Requalifier{
							Qualifier: "b",
							Source: &physical.Requalifier{
								Qualifier: "c",
								Source: &PlaceholderNode{
									Name: "stub",
								},
							},
						},
					},
				},
			},
			want: &physical.Map{
				Expressions: []physical.NamedExpression{physical.NewVariable("expr")},
				Source: &physical.Requalifier{
					Qualifier: "a",
					Source: &PlaceholderNode{
						Name: "stub",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Optimize(context.Background(), []Scenario{MergeRequalifiers}, tt.args.plan); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeRequalifiers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeFilters(t *testing.T) {
	type args struct {
		plan physical.Node
	}
	tests := []struct {
		name string
		args args
		want physical.Node
	}{
		{
			name: "simple single merge",
			args: args{
				plan: &physical.Filter{
					Formula: physical.NewPredicate(
						physical.NewVariable("parent_left"),
						physical.Equal,
						physical.NewVariable("parent_right"),
					),
					Source: &physical.Filter{
						Formula: physical.NewPredicate(
							physical.NewVariable("child_left"),
							physical.NotEqual,
							physical.NewVariable("child_right"),
						),
						Source: &PlaceholderNode{
							Name: "stub",
						},
					},
				},
			},
			want: &physical.Filter{
				Formula: physical.NewAnd(
					physical.NewPredicate(
						physical.NewVariable("parent_left"),
						physical.Equal,
						physical.NewVariable("parent_right"),
					),
					physical.NewPredicate(
						physical.NewVariable("child_left"),
						physical.NotEqual,
						physical.NewVariable("child_right"),
					),
				),
				Source: &PlaceholderNode{
					Name: "stub",
				},
			},
		},
		{
			name: "multi merge",
			args: args{
				plan: &physical.Filter{
					Formula: physical.NewPredicate(
						physical.NewVariable("a_left"),
						physical.Equal,
						physical.NewVariable("a_right"),
					),
					Source: &physical.Filter{
						Formula: physical.NewPredicate(
							physical.NewVariable("b_left"),
							physical.MoreThan,
							physical.NewVariable("b_right"),
						),
						Source: &physical.Filter{
							Formula: physical.NewPredicate(
								physical.NewVariable("c_left"),
								physical.LessThan,
								physical.NewVariable("c_right"),
							),
							Source: &physical.Filter{
								Formula: physical.NewPredicate(
									physical.NewVariable("d_left"),
									physical.NotEqual,
									physical.NewVariable("d_right"),
								),
								Source: &PlaceholderNode{
									Name: "stub",
								},
							},
						},
					},
				},
			},
			want: &physical.Filter{
				Formula: physical.NewAnd(
					physical.NewPredicate(
						physical.NewVariable("a_left"),
						physical.Equal,
						physical.NewVariable("a_right"),
					),
					physical.NewAnd(
						physical.NewPredicate(
							physical.NewVariable("b_left"),
							physical.MoreThan,
							physical.NewVariable("b_right"),
						),
						physical.NewAnd(
							physical.NewPredicate(
								physical.NewVariable("c_left"),
								physical.LessThan,
								physical.NewVariable("c_right"),
							),
							physical.NewPredicate(
								physical.NewVariable("d_left"),
								physical.NotEqual,
								physical.NewVariable("d_right"),
							),
						),
					),
				),
				Source: &PlaceholderNode{
					Name: "stub",
				},
			},
		},
		{
			name: "deeper merge",
			args: args{
				plan: &physical.Map{
					Expressions: []physical.NamedExpression{physical.NewVariable("expr")},
					Source: &physical.Filter{
						Formula: physical.NewPredicate(
							physical.NewVariable("a_left"),
							physical.Equal,
							physical.NewVariable("a_right"),
						),
						Source: &physical.Filter{
							Formula: physical.NewPredicate(
								physical.NewVariable("b_left"),
								physical.MoreThan,
								physical.NewVariable("b_right"),
							),
							Source: &physical.Filter{
								Formula: physical.NewPredicate(
									physical.NewVariable("c_left"),
									physical.LessThan,
									physical.NewVariable("c_right"),
								),
								Source: &PlaceholderNode{
									Name: "stub",
								},
							},
						},
					},
				},
			},
			want: &physical.Map{
				Expressions: []physical.NamedExpression{physical.NewVariable("expr")},
				Source: &physical.Filter{
					Formula: physical.NewAnd(
						physical.NewPredicate(
							physical.NewVariable("a_left"),
							physical.Equal,
							physical.NewVariable("a_right"),
						),
						physical.NewAnd(
							physical.NewPredicate(
								physical.NewVariable("b_left"),
								physical.MoreThan,
								physical.NewVariable("b_right"),
							),
							physical.NewPredicate(
								physical.NewVariable("c_left"),
								physical.LessThan,
								physical.NewVariable("c_right"),
							),
						),
					),
					Source: &PlaceholderNode{
						Name: "stub",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Optimize(context.Background(), []Scenario{MergeFilters}, tt.args.plan); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeFilters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeDataSourceWithRequalifier(t *testing.T) {
	type args struct {
		plan physical.Node
	}
	tests := []struct {
		name string
		args args
		want physical.Node
	}{
		{
			name: "simple single merge",
			args: args{
				plan: &physical.Requalifier{
					Qualifier: "a",
					Source: &physical.DataSourceBuilder{
						Executor:    nil,
						PrimaryKeys: []octosql.VariableName{octosql.NewVariableName("a")},
						AvailableFilters: map[physical.FieldType]map[physical.Relation]struct{}{
							physical.Primary: {
								physical.Equal:    struct{}{},
								physical.NotEqual: struct{}{},
							},
							physical.Secondary: {
								physical.Equal:    struct{}{},
								physical.NotEqual: struct{}{},
								physical.MoreThan: struct{}{},
								physical.LessThan: struct{}{},
							},
						},
						Filter: physical.NewAnd(
							physical.NewConstant(true),
							physical.NewConstant(false),
						),
						Alias: "b",
					},
				},
			},
			want: &physical.DataSourceBuilder{
				Executor:    nil,
				PrimaryKeys: []octosql.VariableName{octosql.NewVariableName("a")},
				AvailableFilters: map[physical.FieldType]map[physical.Relation]struct{}{
					physical.Primary: {
						physical.Equal:    struct{}{},
						physical.NotEqual: struct{}{},
					},
					physical.Secondary: {
						physical.Equal:    struct{}{},
						physical.NotEqual: struct{}{},
						physical.MoreThan: struct{}{},
						physical.LessThan: struct{}{},
					},
				},
				Filter: physical.NewAnd(
					physical.NewConstant(true),
					physical.NewConstant(false),
				),
				Alias: "a",
			},
		},
		{
			name: "multi merge",
			args: args{
				plan: &physical.Requalifier{
					Qualifier: "a",
					Source: &physical.Requalifier{
						Qualifier: "b",
						Source: &physical.DataSourceBuilder{
							Executor:    nil,
							PrimaryKeys: []octosql.VariableName{octosql.NewVariableName("a")},
							AvailableFilters: map[physical.FieldType]map[physical.Relation]struct{}{
								physical.Primary: {
									physical.Equal:    struct{}{},
									physical.NotEqual: struct{}{},
								},
								physical.Secondary: {
									physical.Equal:    struct{}{},
									physical.NotEqual: struct{}{},
									physical.MoreThan: struct{}{},
									physical.LessThan: struct{}{},
								},
							},
							Filter: physical.NewAnd(
								physical.NewConstant(true),
								physical.NewConstant(false),
							),
							Alias: "c",
						},
					},
				},
			},
			want: &physical.DataSourceBuilder{
				Executor:    nil,
				PrimaryKeys: []octosql.VariableName{octosql.NewVariableName("a")},
				AvailableFilters: map[physical.FieldType]map[physical.Relation]struct{}{
					physical.Primary: {
						physical.Equal:    struct{}{},
						physical.NotEqual: struct{}{},
					},
					physical.Secondary: {
						physical.Equal:    struct{}{},
						physical.NotEqual: struct{}{},
						physical.MoreThan: struct{}{},
						physical.LessThan: struct{}{},
					},
				},
				Filter: physical.NewAnd(
					physical.NewConstant(true),
					physical.NewConstant(false),
				),
				Alias: "a",
			},
		},
		{
			name: "simple single merge",
			args: args{
				plan: &physical.Map{
					Expressions: []physical.NamedExpression{physical.NewVariable("expr")},
					Source: &physical.Requalifier{
						Qualifier: "a",
						Source: &physical.DataSourceBuilder{
							Executor:    nil,
							PrimaryKeys: []octosql.VariableName{octosql.NewVariableName("a")},
							AvailableFilters: map[physical.FieldType]map[physical.Relation]struct{}{
								physical.Primary: {
									physical.Equal:    struct{}{},
									physical.NotEqual: struct{}{},
								},
								physical.Secondary: {
									physical.Equal:    struct{}{},
									physical.NotEqual: struct{}{},
									physical.MoreThan: struct{}{},
									physical.LessThan: struct{}{},
								},
							},
							Filter: physical.NewAnd(
								physical.NewConstant(true),
								physical.NewConstant(false),
							),
							Alias: "b",
						},
					},
				},
			},
			want: &physical.Map{
				Expressions: []physical.NamedExpression{physical.NewVariable("expr")},
				Source: &physical.DataSourceBuilder{
					Executor:    nil,
					PrimaryKeys: []octosql.VariableName{octosql.NewVariableName("a")},
					AvailableFilters: map[physical.FieldType]map[physical.Relation]struct{}{
						physical.Primary: {
							physical.Equal:    struct{}{},
							physical.NotEqual: struct{}{},
						},
						physical.Secondary: {
							physical.Equal:    struct{}{},
							physical.NotEqual: struct{}{},
							physical.MoreThan: struct{}{},
							physical.LessThan: struct{}{},
						},
					},
					Filter: physical.NewAnd(
						physical.NewConstant(true),
						physical.NewConstant(false),
					),
					Alias: "a",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Optimize(context.Background(), []Scenario{MergeDataSourceBuilderWithRequalifier}, tt.args.plan); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeDataSourceWithRequalifier() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeDataSourceWithFilter(t *testing.T) {
	type args struct {
		plan physical.Node
	}
	tests := []struct {
		name string
		args args
		want physical.Node
	}{
		{
			name: "simple single merge",
			args: args{
				plan: &physical.Filter{
					Formula: physical.NewPredicate(
						physical.NewVariable("a.name"),
						physical.Equal,
						physical.NewVariable("a.surname"),
					),
					Source: &physical.DataSourceBuilder{
						Executor:    nil,
						PrimaryKeys: []octosql.VariableName{},
						AvailableFilters: map[physical.FieldType]map[physical.Relation]struct{}{
							physical.Primary: {
								physical.Equal:    struct{}{},
								physical.NotEqual: struct{}{},
								physical.MoreThan: struct{}{},
								physical.LessThan: struct{}{},
							},
							physical.Secondary: {
								physical.Equal: struct{}{},
							},
						},
						Filter: physical.NewConstant(true),
						Alias:  "a",
					},
				},
			},
			want: &physical.DataSourceBuilder{
				Executor:    nil,
				PrimaryKeys: []octosql.VariableName{},
				AvailableFilters: map[physical.FieldType]map[physical.Relation]struct{}{
					physical.Primary: {
						physical.Equal:    struct{}{},
						physical.NotEqual: struct{}{},
						physical.MoreThan: struct{}{},
						physical.LessThan: struct{}{},
					},
					physical.Secondary: {
						physical.Equal: struct{}{},
					},
				},
				Filter: physical.NewAnd(
					physical.NewPredicate(
						physical.NewVariable("a.name"),
						physical.Equal,
						physical.NewVariable("a.surname"),
					),
					physical.NewConstant(true),
				),
				Alias: "a",
			},
		},
		{
			name: "not mergable",
			args: args{
				plan: &physical.Filter{
					Formula: physical.NewPredicate(
						physical.NewFunctionExpression("test", []physical.Expression{physical.NewVariable("a.name")}),
						physical.Equal,
						physical.NewVariable("b.test"),
					),
					Source: &physical.DataSourceBuilder{
						Executor:    nil,
						PrimaryKeys: []octosql.VariableName{},
						AvailableFilters: map[physical.FieldType]map[physical.Relation]struct{}{
							physical.Primary: {
								physical.Equal:    struct{}{},
								physical.NotEqual: struct{}{},
								physical.MoreThan: struct{}{},
								physical.LessThan: struct{}{},
							},
							physical.Secondary: {
								physical.Equal: struct{}{},
							},
						},
						Filter: physical.NewConstant(true),
						Alias:  "a",
					},
				},
			},
			want: &physical.Filter{
				Formula: physical.NewPredicate(
					physical.NewFunctionExpression("test", []physical.Expression{physical.NewVariable("a.name")}),
					physical.Equal,
					physical.NewVariable("b.test"),
				),
				Source: &physical.DataSourceBuilder{
					Executor:    nil,
					PrimaryKeys: []octosql.VariableName{},
					AvailableFilters: map[physical.FieldType]map[physical.Relation]struct{}{
						physical.Primary: {
							physical.Equal:    struct{}{},
							physical.NotEqual: struct{}{},
							physical.MoreThan: struct{}{},
							physical.LessThan: struct{}{},
						},
						physical.Secondary: {
							physical.Equal: struct{}{},
						},
					},
					Filter: physical.NewConstant(true),
					Alias:  "a",
				},
			},
		},
		{
			name: "not mergable",
			args: args{
				plan: &physical.Filter{
					Formula: physical.NewPredicate(
						physical.NewFunctionExpression("test", []physical.Expression{physical.NewVariable("a.name")}),
						physical.Equal,
						physical.NewVariable("a.test"),
					),
					Source: &physical.DataSourceBuilder{
						Executor:    nil,
						PrimaryKeys: []octosql.VariableName{},
						AvailableFilters: map[physical.FieldType]map[physical.Relation]struct{}{
							physical.Primary: {
								physical.Equal:    struct{}{},
								physical.NotEqual: struct{}{},
								physical.MoreThan: struct{}{},
								physical.LessThan: struct{}{},
							},
							physical.Secondary: {
								physical.Equal: struct{}{},
							},
						},
						Filter: physical.NewConstant(true),
						Alias:  "a",
					},
				},
			},
			want: &physical.Filter{
				Formula: physical.NewPredicate(
					physical.NewFunctionExpression("test", []physical.Expression{physical.NewVariable("a.name")}),
					physical.Equal,
					physical.NewVariable("a.test"),
				),
				Source: &physical.DataSourceBuilder{
					Executor:    nil,
					PrimaryKeys: []octosql.VariableName{},
					AvailableFilters: map[physical.FieldType]map[physical.Relation]struct{}{
						physical.Primary: {
							physical.Equal:    struct{}{},
							physical.NotEqual: struct{}{},
							physical.MoreThan: struct{}{},
							physical.LessThan: struct{}{},
						},
						physical.Secondary: {
							physical.Equal: struct{}{},
						},
					},
					Filter: physical.NewConstant(true),
					Alias:  "a",
				},
			},
		},
		{
			name: "mergable",
			args: args{
				plan: &physical.Filter{
					Formula: physical.NewPredicate(
						physical.NewFunctionExpression("test", []physical.Expression{physical.NewVariable("b.name")}),
						physical.Equal,
						physical.NewVariable("a.test"),
					),
					Source: &physical.DataSourceBuilder{
						Executor:    nil,
						PrimaryKeys: []octosql.VariableName{},
						AvailableFilters: map[physical.FieldType]map[physical.Relation]struct{}{
							physical.Primary: {
								physical.Equal:    struct{}{},
								physical.NotEqual: struct{}{},
								physical.MoreThan: struct{}{},
								physical.LessThan: struct{}{},
							},
							physical.Secondary: {
								physical.Equal: struct{}{},
							},
						},
						Filter: physical.NewConstant(true),
						Alias:  "a",
					},
				},
			},
			want: &physical.DataSourceBuilder{
				Executor:    nil,
				PrimaryKeys: []octosql.VariableName{},
				AvailableFilters: map[physical.FieldType]map[physical.Relation]struct{}{
					physical.Primary: {
						physical.Equal:    struct{}{},
						physical.NotEqual: struct{}{},
						physical.MoreThan: struct{}{},
						physical.LessThan: struct{}{},
					},
					physical.Secondary: {
						physical.Equal: struct{}{},
					},
				},
				Filter: physical.NewAnd(
					physical.NewPredicate(
						physical.NewFunctionExpression("test", []physical.Expression{physical.NewVariable("b.name")}),
						physical.Equal,
						physical.NewVariable("a.test"),
					),
					physical.NewConstant(true),
				),
				Alias: "a",
			},
		},
		{
			name: "simple partial merge",
			args: args{
				plan: &physical.Filter{
					Formula: physical.NewAnd(
						physical.NewPredicate(
							physical.NewVariable("a.name"),
							physical.Equal,
							physical.NewVariable("a.surname"),
						),
						physical.NewPredicate(
							physical.NewVariable("b.test"),
							physical.MoreThan,
							physical.NewVariable("a.surname"),
						),
					),

					Source: &physical.DataSourceBuilder{
						Executor:    nil,
						PrimaryKeys: []octosql.VariableName{"a.name"},
						AvailableFilters: map[physical.FieldType]map[physical.Relation]struct{}{
							physical.Primary: {
								physical.Equal:    struct{}{},
								physical.NotEqual: struct{}{},
								physical.MoreThan: struct{}{},
								physical.LessThan: struct{}{},
							},
							physical.Secondary: {
								physical.Equal: struct{}{},
							},
						},
						Filter: physical.NewConstant(true),
						Alias:  "a",
					},
				},
			},
			want: &physical.Filter{
				Formula: physical.NewPredicate(
					physical.NewVariable("b.test"),
					physical.MoreThan,
					physical.NewVariable("a.surname"),
				),
				Source: &physical.DataSourceBuilder{
					Executor:    nil,
					PrimaryKeys: []octosql.VariableName{"a.name"},
					AvailableFilters: map[physical.FieldType]map[physical.Relation]struct{}{
						physical.Primary: {
							physical.Equal:    struct{}{},
							physical.NotEqual: struct{}{},
							physical.MoreThan: struct{}{},
							physical.LessThan: struct{}{},
						},
						physical.Secondary: {
							physical.Equal: struct{}{},
						},
					},
					Filter: physical.NewAnd(
						physical.NewPredicate(
							physical.NewVariable("a.name"),
							physical.Equal,
							physical.NewVariable("a.surname"),
						),
						physical.NewConstant(true),
					),
					Alias: "a",
				},
			},
		},
		{
			name: "simple partial merge",
			args: args{
				plan: &physical.Filter{
					Formula: physical.NewAnd(
						physical.NewPredicate(
							physical.NewVariable("a.name"),
							physical.Equal,
							physical.NewVariable("a.surname"),
						),
						physical.NewPredicate(
							physical.NewVariable("b.test"),
							physical.MoreThan,
							physical.NewVariable("b.test2"),
						),
					),

					Source: &physical.DataSourceBuilder{
						Executor:    nil,
						PrimaryKeys: []octosql.VariableName{"a.name"},
						AvailableFilters: map[physical.FieldType]map[physical.Relation]struct{}{
							physical.Primary: {
								physical.Equal:    struct{}{},
								physical.NotEqual: struct{}{},
								physical.MoreThan: struct{}{},
								physical.LessThan: struct{}{},
							},
							physical.Secondary: {
								physical.Equal: struct{}{},
							},
						},
						Filter: physical.NewConstant(true),
						Alias:  "a",
					},
				},
			},
			want: &physical.Filter{
				Formula: physical.NewPredicate(
					physical.NewVariable("b.test"),
					physical.MoreThan,
					physical.NewVariable("b.test2"),
				),
				Source: &physical.DataSourceBuilder{
					Executor:    nil,
					PrimaryKeys: []octosql.VariableName{"a.name"},
					AvailableFilters: map[physical.FieldType]map[physical.Relation]struct{}{
						physical.Primary: {
							physical.Equal:    struct{}{},
							physical.NotEqual: struct{}{},
							physical.MoreThan: struct{}{},
							physical.LessThan: struct{}{},
						},
						physical.Secondary: {
							physical.Equal: struct{}{},
						},
					},
					Filter: physical.NewAnd(
						physical.NewPredicate(
							physical.NewVariable("a.name"),
							physical.Equal,
							physical.NewVariable("a.surname"),
						),
						physical.NewConstant(true),
					),
					Alias: "a",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Optimize(context.Background(), []Scenario{MergeDataSourceBuilderWithFilter}, tt.args.plan); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeDataSourceWithFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMultiOptimization(t *testing.T) {
	type args struct {
		plan physical.Node
	}
	tests := []struct {
		name string
		args args
		want physical.Node
	}{
		{
			name: "multiple optimizations",
			args: args{
				plan: &physical.Map{
					Expressions: []physical.NamedExpression{physical.NewVariable("expr")},
					Source: &physical.Requalifier{
						Qualifier: "a",
						Source: &physical.Requalifier{
							Qualifier: "b",
							Source: &physical.Filter{
								Formula: physical.NewConstant(true),
								Source: &physical.Filter{
									Formula: physical.NewConstant(false),
									Source: &physical.Filter{
										Formula: physical.NewConstant(true),
										Source: &physical.Requalifier{
											Qualifier: "a",
											Source: &physical.Requalifier{
												Qualifier: "b",
												Source: &physical.DataSourceBuilder{
													Executor:    nil,
													PrimaryKeys: []octosql.VariableName{octosql.NewVariableName("a")},
													AvailableFilters: map[physical.FieldType]map[physical.Relation]struct{}{
														physical.Primary: {
															physical.Equal:    struct{}{},
															physical.NotEqual: struct{}{},
														},
														physical.Secondary: {
															physical.Equal:    struct{}{},
															physical.NotEqual: struct{}{},
															physical.MoreThan: struct{}{},
															physical.LessThan: struct{}{},
														},
													},
													Filter: physical.NewAnd(
														physical.NewConstant(true),
														physical.NewConstant(false),
													),
													Alias: "c",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: &physical.Map{
				Expressions: []physical.NamedExpression{physical.NewVariable("expr")},
				Source: &physical.Requalifier{
					Qualifier: "a",
					Source: &physical.Filter{
						Formula: physical.NewAnd(
							physical.NewConstant(true),
							physical.NewAnd(
								physical.NewConstant(false),
								physical.NewConstant(true),
							),
						),
						Source: &physical.DataSourceBuilder{
							Executor:    nil,
							PrimaryKeys: []octosql.VariableName{octosql.NewVariableName("a")},
							AvailableFilters: map[physical.FieldType]map[physical.Relation]struct{}{
								physical.Primary: {
									physical.Equal:    struct{}{},
									physical.NotEqual: struct{}{},
								},
								physical.Secondary: {
									physical.Equal:    struct{}{},
									physical.NotEqual: struct{}{},
									physical.MoreThan: struct{}{},
									physical.LessThan: struct{}{},
								},
							},
							Filter: physical.NewAnd(
								physical.NewConstant(true),
								physical.NewConstant(false),
							),
							Alias: "a",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Optimize(
				context.Background(),
				[]Scenario{
					MergeRequalifiers,
					MergeFilters,
					MergeDataSourceBuilderWithRequalifier,
				},
				tt.args.plan,
			); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeDataSourceWithRequalifier() = %v, want %v", got, tt.want)
			}
		})
	}
}
