package planner_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/n9te9/federation-gateway/federation/graph"
	"github.com/n9te9/federation-gateway/federation/planner"
	"github.com/n9te9/goliteql/query"
)

func TestPlanner_Plan(t *testing.T) {
	tests := []struct {
		name       string
		doc        *query.Document
		superGraph *graph.SuperGraph
		want       *planner.Plan
		wantErr    error
	}{
		{
			name: "Happy case: Plan simple query",
			doc: func() *query.Document {
				lexer := query.NewLexer()
				parser := query.NewParser(lexer)
				doc, err := parser.Parse([]byte(`
					query {
						products {
							upc
							name
							price
						}
					}
				`))
				if err != nil {
					t.Fatal(err)
				}

				return doc
			}(),
			superGraph: func() *graph.SuperGraph {
				sdl := `type Query {
					products: [Product]
				}

				type Product {
					upc: String!
					name: String
					price: Int
				}`

				superGraph, err := graph.NewSuperGraphFromBytes([]byte(sdl))
				if err != nil {
					t.Fatalf("failed to parse root schema: %v", err)
				}

				return superGraph
			}(),
			want: &planner.Plan{
				Steps: []*planner.Step{
					{
						SubGraph:  nil,
						DependsOn: nil,
						Status:    planner.Pending,
						Err:       nil,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planner := planner.NewPlanner(tt.superGraph)
			got, err := planner.Plan(tt.doc)
			if (err != nil) != (tt.wantErr != nil) {
				t.Fatalf("Planner.Plan() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}

			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("Planner.Plan() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
