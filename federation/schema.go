package federation

import "github.com/n9te9/federation-gateway/federation/graph"

type Schema struct {
	SuperGraph *graph.SuperGraph
}

func NewSchema(superGraph *graph.SuperGraph, name, rootHost string) *Schema {
	rootGraph, err := graph.NewSubGraph(name, []byte(superGraph.SDL), rootHost)
	if err != nil {
		panic(err)
	}
	superGraph.RootGraph = rootGraph

	return &Schema{
		SuperGraph: superGraph,
	}
}
