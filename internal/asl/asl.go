package asl

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/awalterschulze/gographviz"
	"github.com/serenize/snaker"
	"gopkg.in/yaml.v3"
)

// https://states-language.net/spec.html
type StateMachine struct {
	StartAt string            `json:"StartAt,omitempty" yaml:"StartAt,omitempty"`
	States  map[string]*State `json:"States,omitempty" yaml:"States,omitempty"`
}

type State struct {
	Type     string          `json:"Type,omitempty" yaml:"Type,omitempty"`
	Next     string          `json:"Next,omitempty" yaml:"Next,omitempty"`
	End      bool            `json:"End,omitempty" yaml:"End,omitempty"`
	Default  string          `json:"Default,omitempty" yaml:"Default,omitempty"`
	Catch    []*Catch        `json:"Catch,omitempty" yaml:"Catch,omitempty"`
	Choices  []*Choice       `json:"Choices,omitempty" yaml:"Choices,omitempty"`
	Iterator *StateMachine   `json:"Iterator,omitempty" yaml:"Iterator,omitempty"`
	Branches []*StateMachine `json:"Branches,omitempty" yaml:"Branches,omitempty"`
}

type Choice struct {
	Next string `json:"Next,omitempty" yaml:"Next,omitempty"`
}

type Catch struct {
	ErrorEquals []string `json:"ErrorEquals,omitempty" yaml:"ErrorEquals,omitempty"`
	Next        string   `json:"Next,omitempty" yaml:"Next,omitempty"`
}

func Parse(str string) (*StateMachine, error) {
	data := []byte(str)
	var stateMachine StateMachine
	if err := json.Unmarshal(data, &stateMachine); err == nil {
		return &stateMachine, nil
	}
	if err := yaml.Unmarshal(data, &stateMachine); err == nil {
		return &stateMachine, nil
	}
	return nil, errors.New("invalid format can not parse as yaml or json")
}

func (s *StateMachine) MarshalDOT(name string) ([]byte, error) {
	g := gographviz.NewGraph()
	graphName := snaker.CamelToSnake(name)
	nodeAttrs := make(map[string]string)
	edgeAttrs := make(map[string]string)
	edgeAttrs["arrowhead"] = "vee"
	if err := g.SetName(graphName); err != nil {
		return nil, err
	}
	if err := g.SetDir(true); err != nil {
		return nil, err
	}
	if err := s.toGraph(g, graphName, nodeAttrs, edgeAttrs); err != nil {
		return nil, err
	}
	return []byte(g.String()), nil
}

func (s *StateMachine) toGraph(g *gographviz.Graph, graphName string, nodeAttrs map[string]string, edgeAttrs map[string]string) error {
	if err := addGraphNode(g, graphName, graphName+"_start", nodeAttrs); err != nil {
		return err
	}
	if err := addGraphNode(g, graphName, graphName+"_end", nodeAttrs); err != nil {
		return err
	}
	if s.StartAt != "" {
		if err := addGraphEdge(g, graphName+"_start", s.StartAt, "", edgeAttrs); err != nil {
			return err
		}
	} else {
		if err := addGraphEdge(g, graphName+"_start", graphName+"_end", "", edgeAttrs); err != nil {
			return err
		}
	}
	for stateName, state := range s.States {
		if err := state.toGraph(g, graphName, stateName, nodeAttrs, edgeAttrs); err != nil {
			return err
		}
	}
	return nil
}

func (s *StateMachine) toSubGraph(g *gographviz.Graph, parentGraphName string, subGraphName string, stateName string, nodeAttrs map[string]string, edgeAttrs map[string]string) error {
	if err := g.AddSubGraph(parentGraphName, subGraphName, nil); err != nil {
		return err
	}

	if err := s.toGraph(g, subGraphName, nodeAttrs, edgeAttrs); err != nil {
		return err
	}

	orgStyle, needRestoreStyle := edgeAttrs["style"]
	edgeAttrs["style"] = "dotted"
	if err := addGraphEdge(g, stateName, subGraphName+"_start", "", edgeAttrs); err != nil {
		return err
	}
	if needRestoreStyle {
		edgeAttrs["style"] = orgStyle
	} else {
		delete(edgeAttrs, "style")
	}
	return nil
}

func (s *State) toGraph(g *gographviz.Graph, graphName string, stateName string, nodeAttrs map[string]string, edgeAttrs map[string]string) error {
	if err := addGraphNode(g, graphName, stateName, nodeAttrs); err != nil {
		return err
	}

	if s.Next != "" {
		if err := addGraphEdge(g, stateName, s.Next, "", edgeAttrs); err != nil {
			return err
		}
	}
	if s.End {
		if err := addGraphEdge(g, stateName, graphName+"_end", "", edgeAttrs); err != nil {
			return err
		}
	}
	for i, catch := range s.Catch {
		if catch.Next != "" {
			if err := addGraphEdge(g, stateName, catch.Next, fmt.Sprintf("catch #%d", i+1), edgeAttrs); err != nil {
				return err
			}
		}
	}

	switch s.Type {
	case "Choice":
		if s.Default != "" {
			if err := addGraphEdge(g, stateName, s.Default, "Default", edgeAttrs); err != nil {
				return err
			}
		}
		for i, choice := range s.Choices {
			if choice.Next != "" {
				if err := addGraphEdge(g, stateName, choice.Next, fmt.Sprintf("rule #%d", i+1), edgeAttrs); err != nil {
					return err
				}
			}
		}
	case "Succeed", "Fail":
		if err := addGraphEdge(g, stateName, graphName+"_end", "", edgeAttrs); err != nil {
			return err
		}

	case "Map":
		subGraphName := "cluster_" + snaker.CamelToSnake(stateName)
		if s.Iterator == nil {
			s.Iterator = &StateMachine{}
		}
		if err := s.Iterator.toSubGraph(g, graphName, subGraphName, stateName, nodeAttrs, edgeAttrs); err != nil {
			return err
		}
	case "Parallel":
		subGraphPrefix := "cluster_" + snaker.CamelToSnake(stateName)
		for j, branch := range s.Branches {
			if branch == nil {
				branch = &StateMachine{}
			}
			subGraphName := fmt.Sprintf("%s_branch%d", subGraphPrefix, j+1)
			if err := branch.toSubGraph(g, graphName, subGraphName, stateName, nodeAttrs, edgeAttrs); err != nil {
				return err
			}
		}
	}
	return nil
}

func addGraphNode(g *gographviz.Graph, graphName string, nodeName string, nodeAttrs map[string]string) error {
	if nodeName == graphName+"_start" || nodeName == graphName+"_end" {
		nodeAttrs["shape"] = `"ellipse"`
		nodeAttrs["style"] = `"filled"`
	} else {
		nodeAttrs["shape"] = `"box"`
		nodeAttrs["style"] = `"rounded,filled"`
	}
	return g.AddNode(graphName, nodeName, nodeAttrs)
}

func addGraphEdge(g *gographviz.Graph, src string, dst string, label string, edgeAttrs map[string]string) error {
	if label == "" {
		delete(edgeAttrs, "label")
	} else {
		edgeAttrs["label"] = `"` + label + `"`
	}
	return g.AddEdge(src, dst, true, edgeAttrs)
}
