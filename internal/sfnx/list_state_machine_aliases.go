package sfnx

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sfn"
)

type ListStateMachineAliasesAPIClient interface {
	ListStateMachineAliases(ctx context.Context, params *sfn.ListStateMachineAliasesInput, optFns ...func(*sfn.Options)) (*sfn.ListStateMachineAliasesOutput, error)
}

type ListStateMachineAliasesPaginator struct {
	client    ListStateMachineAliasesAPIClient
	params    *sfn.ListStateMachineAliasesInput
	nextToken *string
	firstPage bool
}

func NewListStateMachineAliasesPaginator(client ListStateMachineAliasesAPIClient, params *sfn.ListStateMachineAliasesInput) *ListStateMachineAliasesPaginator {
	if params == nil {
		params = &sfn.ListStateMachineAliasesInput{}
	}

	return &ListStateMachineAliasesPaginator{
		client:    client,
		params:    params,
		firstPage: true,
	}
}

func (p *ListStateMachineAliasesPaginator) HasMorePages() bool {
	return p.firstPage || p.nextToken != nil
}

func (p *ListStateMachineAliasesPaginator) NextPage(ctx context.Context, optFns ...func(*sfn.Options)) (*sfn.ListStateMachineAliasesOutput, error) {
	if !p.HasMorePages() {
		return nil, fmt.Errorf("no more pages available")
	}

	params := *p.params
	params.NextToken = p.nextToken

	result, err := p.client.ListStateMachineAliases(ctx, &params, optFns...)
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
