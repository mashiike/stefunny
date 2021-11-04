package asl_test

import (
	"testing"

	"github.com/mashiike/stefunny/asl"
	"github.com/mashiike/stefunny/internal/testutils"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	cases := []struct {
		path     string
		expected *asl.StateMachine
	}{
		{
			path: "../testdata/hello_world.asl.json",
			expected: &asl.StateMachine{
				StartAt: "Hello",
				States: map[string]*asl.State{
					"Hello": {
						Type: "Pass",
						Next: "World",
					},
					"World": {
						Type: "Pass",
						End:  true,
					},
				},
			},
		},
		{
			path: "../testdata/hello_world.asl.yaml",
			expected: &asl.StateMachine{
				StartAt: "Hello",
				States: map[string]*asl.State{
					"Hello": {
						Type: "Pass",
						Next: "World",
					},
					"World": {
						Type: "Pass",
						End:  true,
					},
				},
			},
		},
		{
			path: "../testdata/workflow1.asl.json",
			expected: &asl.StateMachine{
				StartAt: "Choice",
				States: map[string]*asl.State{
					"Choice": {
						Type: "Choice",
						Choices: []*asl.Choice{
							{
								Next: "Pass",
							},
							{
								Next: "Map",
							},
						},
						Default: "Default",
					},
					"Default": {
						Type: "Pass",
						Next: "Pass",
					},
					"Fail": {
						Type: "Fail",
					},
					"Parallel": {
						Type: "Parallel",
						Next: "Success",
						Branches: []*asl.StateMachine{
							{
								StartAt: "pass2",
								States: map[string]*asl.State{
									"pass2": {
										Type: "Pass",
										End:  true,
									},
								},
							},
							{
								StartAt: "pass3",
								States: map[string]*asl.State{
									"pass3": {
										Type: "Pass",
										End:  true,
									},
								},
							},
						},
					},
					"Map": {
						Type: "Map",
						Iterator: &asl.StateMachine{
							StartAt: "Map1",
							States: map[string]*asl.State{
								"Map1": {
									Type: "Pass",
									End:  true,
								},
							},
						},
						Catch: []*asl.Catch{
							{
								ErrorEquals: []string{"States.ALL"},
								Next:        "Pass",
							},
						},
						Next: "Wait",
					},
					"Success": {
						Type: "Succeed",
					},
					"Pass": {
						Type: "Pass",
						Next: "Parallel",
					},
					"Wait": {
						Type: "Wait",
						Next: "Fail",
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			stateMachine, err := asl.Parse(testutils.LoadString(t, c.path))
			require.NoError(t, err)
			require.EqualValues(t, c.expected, stateMachine)
		})
	}
}
