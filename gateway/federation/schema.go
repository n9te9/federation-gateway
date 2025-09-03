package federation

import (
	"context"

	"github.com/n9te9/goliteql/query"
	"github.com/n9te9/goliteql/schema"
)

type SuperGraph struct {
	Schema    *schema.Schema
	SubGraphs []*SubGraph
}

type SubGraph struct {
	Name         string
	Schema       *schema.Schema
	SDL          string
	Host         string
	IsIntegrated bool
}

func NewSuperGraph(root *schema.Schema, subGraphs []*SubGraph) *SuperGraph {
	return &SuperGraph{
		Schema:    root,
		SubGraphs: subGraphs,
	}
}

func NewSuperGraphFromBytes(src []byte) (*SuperGraph, error) {
	schema, err := schema.NewParser(schema.NewLexer()).Parse(src)
	if err != nil {
		return nil, err
	}

	return &SuperGraph{
		Schema:    schema,
		SubGraphs: make([]*SubGraph, 0),
	}, nil
}

func (sg *SuperGraph) Merge() error {
	for _, subGraph := range sg.SubGraphs {
		if !subGraph.IsIntegrated {
			sg.registerExtentions(subGraph.Schema)
			newSchema, err := sg.Schema.Merge()
			if err != nil {
				return err
			}

			sg.Schema = newSchema
			subGraph.IsIntegrated = true
		}
	}

	return nil
}

func (sg *SuperGraph) registerExtentions(subGraphSchema *schema.Schema) {
	sg.Schema.Extends = append(sg.Schema.Extends, subGraphSchema.Extends...)
}

func (sg *SuperGraph) Execute(ctx context.Context, plan *Plan) (any, error) {

	return nil, nil
}

type Plan struct {
	Steps []*Step
}

type Step struct {
	SubGraph  *SubGraph
	DependsOn []*Step

	Status StepStatus
	Err    error
}

type StepStatus int

const (
	Pending StepStatus = iota
	Running
	Completed
	Failed
)

func (sg *SubGraph) Plan(document *query.Document) *Plan {
	return nil
}

func NewSubGraph(name string, src []byte, host string) (*SubGraph, error) {
	schema, err := schema.NewParser(schema.NewLexer()).Parse(src)
	if err != nil {
		return nil, err
	}

	return &SubGraph{
		Name:         name,
		Schema:       schema,
		Host:         host,
		IsIntegrated: false,
	}, nil
}
