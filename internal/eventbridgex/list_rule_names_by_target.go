package eventbridgex

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
)

type ListRuleNamesByTargetAPIClient interface {
	ListRuleNamesByTarget(ctx context.Context, params *eventbridge.ListRuleNamesByTargetInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListRuleNamesByTargetOutput, error)
}

type ListRuleNamesByTargetPaginator struct {
	client    ListRuleNamesByTargetAPIClient
	params    *eventbridge.ListRuleNamesByTargetInput
	nextToken *string
	firstPage bool
}

func NewListRuleNamesByTargetPaginator(client ListRuleNamesByTargetAPIClient, params *eventbridge.ListRuleNamesByTargetInput) *ListRuleNamesByTargetPaginator {
	if params == nil {
		params = &eventbridge.ListRuleNamesByTargetInput{}
	}

	return &ListRuleNamesByTargetPaginator{
		client:    client,
		params:    params,
		firstPage: true,
	}
}

func (p *ListRuleNamesByTargetPaginator) HasMorePages() bool {
	return p.firstPage || p.nextToken != nil
}

func (p *ListRuleNamesByTargetPaginator) NextPage(ctx context.Context, optFns ...func(*eventbridge.Options)) (*eventbridge.ListRuleNamesByTargetOutput, error) {
	if !p.HasMorePages() {
		return nil, fmt.Errorf("no more pages available")
	}

	params := *p.params
	params.NextToken = p.nextToken

	result, err := p.client.ListRuleNamesByTarget(ctx, &params, optFns...)
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
