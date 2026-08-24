package stefunny

import (
	"context"
	"fmt"
	"sync"
	"text/template"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	jsonnet "github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
)

// STSClient is the subset of the STS API used to resolve caller_identity.
//
//go:generate go tool mockgen -source=$GOFILE -destination=./mock/$GOFILE -package=mock
type STSClient interface {
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

// callerIdentity memoizes the result of GetCallerIdentity so that it is
// resolved at most once per Load call, no matter how many times
// jsonnet/template evaluation calls the caller_identity function.
type callerIdentity struct {
	mu     sync.Mutex
	data   map[string]any
	cfg    *Config
	client STSClient
}

// newCallerIdentity is created fresh by ConfigLoader.Load for each call, so
// its memoized data can never leak into an unrelated Load call.
func newCallerIdentity(cfg *Config, client STSClient) *callerIdentity {
	return &callerIdentity{cfg: cfg, client: client}
}

// Get resolves the caller identity, memoizing the result.
//
// If client is set it is used directly; otherwise Get builds its own
// aws.Config independently of cfg.LoadAWSConfig's shared cache. This matters
// because Get can run during the config's own jsonnet evaluation, before
// decode, when cfg.AWSRegion is not yet settled: caching that under-specified
// aws.Config into cfg's shared cache would make later deploys use the wrong
// region.
func (c *callerIdentity) Get(ctx context.Context) (map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data != nil {
		return c.data, nil
	}
	output, err := c.resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get caller identity: %w", err)
	}
	account := coalesce(output.Account)
	arn := coalesce(output.Arn)
	userID := coalesce(output.UserId)
	if account == "" || arn == "" || userID == "" {
		return nil, fmt.Errorf("caller identity response is missing Account, Arn or UserId")
	}
	c.data = map[string]any{
		"Account": account,
		"Arn":     arn,
		"UserId":  userID,
	}
	return c.data, nil
}

func (c *callerIdentity) resolve(ctx context.Context) (*sts.GetCallerIdentityOutput, error) {
	if c.client != nil {
		return c.client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	}
	opts := []func(*awsconfig.LoadOptions) error{}
	if c.cfg.AWSRegion != "" {
		opts = append(opts, awsconfig.WithRegion(c.cfg.AWSRegion))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}
	stsClient := c.cfg.NewStsClientFromConfig(awsCfg)
	return stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
}

// JsonnetNativeFunc returns the std.native('caller_identity') definition for ctx.
func (c *callerIdentity) JsonnetNativeFunc(ctx context.Context) *jsonnet.NativeFunction {
	return &jsonnet.NativeFunction{
		Name:   "caller_identity",
		Params: ast.Identifiers{},
		Func: func(_ []interface{}) (interface{}, error) {
			return c.Get(ctx)
		},
	}
}

// FuncMap returns the text/template caller_identity function for ctx.
func (c *callerIdentity) FuncMap(ctx context.Context) template.FuncMap {
	return template.FuncMap{
		"caller_identity": func() (any, error) {
			return c.Get(ctx)
		},
	}
}
