package stefunny_test

import (
	"context"
	"testing"

	"github.com/mashiike/stefunny"
	"github.com/stretchr/testify/require"
)

func TestListResourcesFromTFState(t *testing.T) {
	ctx := context.Background()
	orderd, err := stefunny.ListResourcesFromTFState(ctx, "testdata/terraform.tfstate")
	require.NoError(t, err)
	require.EqualValues(t, []string{
		"aws_s3_bucket.hoge.bucket",
		"data.aws_caller_identity.current.account_id",
		"aws_s3_bucket.hoge.arn",
		"aws_iam_role.test.arn",
		"aws_cloudwatch_log_group.test.arn",
	}, orderd.Keys())
	values := orderd.Values()
	require.EqualValues(t, []string{
		"test-hoge",
		"000000000000",
		"arn:aws:s3:::test-hoge",
		"arn:aws:iam::000000000000:role/test",
		"arn:aws:logs:ap-northeast-1:000000000000:log-group:test",
	}, values)
}
