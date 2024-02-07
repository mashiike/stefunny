package sfnx

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sfn"
)

type ListStateMachineVersionsAPIClient interface {
	ListStateMachineVersions(ctx context.Context, params *sfn.ListStateMachineVersionsInput, optFns ...func(*sfn.Options)) (*sfn.ListStateMachineVersionsOutput, error)
}

type ListStateMachineVersionsPaginator struct {
	client    ListStateMachineVersionsAPIClient
	params    *sfn.ListStateMachineVersionsInput
	nextToken *string
	firstPage bool
}

func NewListStateMachineVersionsPaginator(client ListStateMachineVersionsAPIClient, params *sfn.ListStateMachineVersionsInput) *ListStateMachineVersionsPaginator {
	if params == nil {
		params = &sfn.ListStateMachineVersionsInput{}
	}

	return &ListStateMachineVersionsPaginator{
		client:    client,
		params:    params,
		firstPage: true,
	}
}

func (p *ListStateMachineVersionsPaginator) HasMorePages() bool {
	return p.firstPage || p.nextToken != nil
}

func (p *ListStateMachineVersionsPaginator) NextPage(ctx context.Context, optFns ...func(*sfn.Options)) (*sfn.ListStateMachineVersionsOutput, error) {
	if !p.HasMorePages() {
		return nil, fmt.Errorf("no more pages available")
	}

	params := *p.params
	params.NextToken = p.nextToken

	result, err := p.client.ListStateMachineVersions(ctx, &params, optFns...)
	if err != nil {
		return nil, err
	}
	p.firstPage = false

	prevToken := p.nextToken
	p.nextToken = result.NextToken

	if prevToken != nil && p.nextToken != nil && *prevToken == *p.nextToken {
		p.nextToken = nil
	}
	return result, nil
}
