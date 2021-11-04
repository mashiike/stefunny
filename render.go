package stefunny

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/awalterschulze/gographviz"
	"github.com/serenize/snaker"
)

func (app *App) Render(ctx context.Context, opt RenderOption) error {
	def, err := app.cfg.LoadDefinition()
	if err != nil {
		return err
	}
	defMap, err := definitionToMap(def)
	if err != nil {
		return err
	}
	g := gographviz.NewGraph()
	graphName := snaker.CamelToSnake(app.cfg.StateMachine.Name)
	nodeAttrs := make(map[string]string)
	edgeAttrs := make(map[string]string)
	edgeAttrs["arrowhead"] = "vee"
	if err := g.SetName(graphName); err != nil {
		return err
	}
	if err := g.SetDir(true); err != nil {
		return err
	}
	if err := parseDefToGraph(g, graphName, defMap, nodeAttrs, edgeAttrs); err != nil {
		return err
	}
	_, err = io.WriteString(opt.Writer, g.String())
	return err
}

func parseDefToGraph(g *gographviz.Graph, graphName string, def map[string]interface{}, nodeAttrs map[string]string, edgeAttrs map[string]string) error {

	if err := addGraphNode(g, graphName, graphName+"_start", nodeAttrs); err != nil {
		return err
	}
	if err := addGraphNode(g, graphName, graphName+"_end", nodeAttrs); err != nil {
		return err
	}
	s, ok := getStringFromMap(def, "StartAt")
	if ok {
		if err := addGraphEdge(g, graphName+"_start", s, "", edgeAttrs); err != nil {
			return err
		}
	} else {
		if err := addGraphEdge(g, graphName+"_start", graphName+"_end", "", edgeAttrs); err != nil {
			return err
		}
	}

	steps, ok := getMapFromMap(def, "States")
	if !ok {
		return errors.New("map key Steps not found")
	}
	for stepName, fuzzyStep := range steps {
		step, ok := fuzzyStep.(map[string]interface{})
		if !ok {
			continue
		}
		if err := addGraphNode(g, graphName, stepName, nodeAttrs); err != nil {
			return err
		}
		if next, ok := getStringFromMap(step, "Next"); ok {
			if err := addGraphEdge(g, stepName, next, "", edgeAttrs); err != nil {
				return err
			}
		}
		if isEnd, ok := getBoolFromMap(step, "End"); ok && isEnd {
			if err := addGraphEdge(g, stepName, graphName+"_end", "", edgeAttrs); err != nil {
				return err
			}
		}
		if cache, ok := getSliceFromMap(step, "Catch"); ok {
			for i, sub := range cache {
				if subMap, ok := sub.(map[string]interface{}); ok {
					if next, ok := getStringFromMap(subMap, "Next"); ok {
						if err := addGraphEdge(g, stepName, next, fmt.Sprintf("catch #%d", i+1), edgeAttrs); err != nil {
							return err
						}
					}
				}
			}
		}
		if stepType, ok := getStringFromMap(step, "Type"); ok {
			switch stepType {
			case "Choice":
				if defaultNext, ok := getStringFromMap(step, "Default"); ok {
					if err := addGraphEdge(g, stepName, defaultNext, "Default", edgeAttrs); err != nil {
						return err
					}
				}
				if choices, ok := getSliceFromMap(step, "Choices"); ok {
					for i, sub := range choices {
						if subMap, ok := sub.(map[string]interface{}); ok {
							if next, ok := getStringFromMap(subMap, "Next"); ok {
								if err := addGraphEdge(g, stepName, next, fmt.Sprintf("rule #%d", i+1), edgeAttrs); err != nil {
									return err
								}
							}
						}
					}
				}

			case "Succeed", "Fail":
				if err := addGraphEdge(g, stepName, graphName+"_end", "", edgeAttrs); err != nil {
					return err
				}

			case "Map":
				subGraphName := "cluster_" + snaker.CamelToSnake(stepName)
				subGraphMap, ok := getMapFromMap(step, "Iterator")
				if !ok {
					subGraphMap = map[string]interface{}{"Steps": []interface{}{}}
				}
				next, _ := getStringFromMap(step, "Next")
				if err := addSubGraph(g, graphName, subGraphName, stepName, next, subGraphMap, nodeAttrs, edgeAttrs); err != nil {
					return err
				}
			case "Parallel":
				subGraphPrefix := "cluster_" + snaker.CamelToSnake(stepName)
				next, _ := getStringFromMap(step, "Next")
				branches, ok := getSliceFromMap(step, "Branches")
				if ok {
					for j, branch := range branches {
						subGraphMap, ok := branch.(map[string]interface{})
						if !ok {
							subGraphMap = map[string]interface{}{"Steps": []interface{}{}}
						}
						subGraphName := fmt.Sprintf("%s_branch%d", subGraphPrefix, j+1)
						if err := addSubGraph(g, graphName, subGraphName, stepName, next, subGraphMap, nodeAttrs, edgeAttrs); err != nil {
							return err
						}
					}
				}

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

func addSubGraph(g *gographviz.Graph, parentGraphName string, subGraphName string, stepName string, _ string, subDef map[string]interface{}, nodeAttrs map[string]string, edgeAttrs map[string]string) error {
	if err := g.AddSubGraph(parentGraphName, subGraphName, nil); err != nil {
		return err
	}

	if err := parseDefToGraph(g, subGraphName, subDef, nodeAttrs, edgeAttrs); err != nil {
		return err
	}

	orgStyle, needRestoreStyle := edgeAttrs["style"]
	edgeAttrs["style"] = "dotted"
	if err := addGraphEdge(g, stepName, subGraphName+"_start", "", edgeAttrs); err != nil {
		return err
	}
	if needRestoreStyle {
		edgeAttrs["style"] = orgStyle
	} else {
		delete(edgeAttrs, "style")
	}
	return nil
}

func getStringFromMap(m map[string]interface{}, key string) (string, bool) {
	v, ok := m[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func getBoolFromMap(m map[string]interface{}, key string) (bool, bool) {
	v, ok := m[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

func getMapFromMap(m map[string]interface{}, key string) (map[string]interface{}, bool) {
	v, ok := m[key]
	if !ok {
		return nil, false
	}
	s, ok := v.(map[string]interface{})
	return s, ok
}

func getSliceFromMap(m map[string]interface{}, key string) ([]interface{}, bool) {
	v, ok := m[key]
	if !ok {
		return nil, false
	}
	s, ok := v.([]interface{})
	return s, ok
}
