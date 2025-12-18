package planner

import (
	"github.com/n9te9/federation-gateway/federation/graph"
	"github.com/n9te9/goliteql/query"
)

type Planner interface {
	Plan(doc *query.Document) (*Plan, error)
}

type planner struct {
	superGraph *graph.SuperGraph
}

type Step struct {
	SubGraph  *graph.SubGraph
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

func NewPlanner(superGraph *graph.SuperGraph) *planner {
	return &planner{
		superGraph: superGraph,
	}
}

type Plan struct {
	Steps []*Step
}

func (p *planner) Plan(doc *query.Document) (*Plan, error) {
	op := p.superGraph.GetOperation(doc)
	_ = op

	return nil, nil
}
