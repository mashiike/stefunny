package stefunny_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/mashiike/stefunny"
	"github.com/mashiike/stefunny/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestConfigLoadCallerIdentity(t *testing.T) {
	t.Setenv("AWS_REGION", "us-east-1")
	wantRoleArn := "arn:aws:iam::012345678901:role/StepFunctions-Hello-Role"

	syntaxCases := []struct {
		name string
		path string
	}{
		{name: "jsonnetのstd.native経由", path: "testdata/caller_identity.jsonnet"},
		{name: "text_templateのcaller_identity関数経由", path: "testdata/caller_identity.yaml"},
	}
	for _, c := range syntaxCases {
		t.Run(c.name+"でモックの値が反映され、GetCallerIdentityは1回だけ呼ばれる", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			client := mock.NewMockSTSClient(ctrl)
			client.EXPECT().GetCallerIdentity(gomock.Any(), gomock.Any()).Return(&sts.GetCallerIdentityOutput{
				Account: aws.String("012345678901"),
				Arn:     aws.String("arn:aws:iam::012345678901:user/dummy"),
				UserId:  aws.String("AIDADUMMY"),
			}, nil).Times(1)

			l := stefunny.NewConfigLoader(nil, nil)
			l.SetSTSClient(client)
			cfg, err := l.Load(context.Background(), c.path)
			require.NoError(t, err)
			require.Equal(t, wantRoleArn, *cfg.StateMachine.Value.RoleArn)
		})
	}

	t.Run("caller_identityを使わないconfigはSTSクライアント未設定でもロードできる", func(t *testing.T) {
		l := stefunny.NewConfigLoader(nil, nil)
		_, err := l.Load(context.Background(), "testdata/stefunny.yaml")
		require.NoError(t, err)
	})

	t.Run("STSクライアント未注入でも既定resolverがAWS_ENDPOINT_URL_STS経由で解決し、cfg.LoadAWSConfigのキャッシュを汚染しない", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/xml")
			_, _ = w.Write([]byte(`<GetCallerIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
  <GetCallerIdentityResult>
    <Arn>arn:aws:iam::012345678901:user/dummy</Arn>
    <UserId>AIDADUMMY</UserId>
    <Account>012345678901</Account>
  </GetCallerIdentityResult>
</GetCallerIdentityResponse>`))
		}))
		t.Cleanup(server.Close)
		t.Setenv("AWS_ENDPOINT_URL_STS", server.URL)
		t.Setenv("AWS_ACCESS_KEY_ID", "dummy")
		t.Setenv("AWS_SECRET_ACCESS_KEY", "dummy")
		t.Setenv("AWS_SESSION_TOKEN", "")
		t.Setenv("AWS_PROFILE", "")
		t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
		t.Setenv("AWS_REGION", "us-east-1")

		l := stefunny.NewConfigLoader(nil, nil)
		ctx := context.Background()
		cfg, err := l.Load(ctx, "testdata/caller_identity_region.jsonnet")
		require.NoError(t, err)
		require.Equal(t, wantRoleArn, *cfg.StateMachine.Value.RoleArn)

		awsCfg, err := cfg.LoadAWSConfig(ctx)
		require.NoError(t, err)
		require.Equal(t, "ap-northeast-1", awsCfg.Region, "cfg.LoadAWSConfig's cache must reflect the config's own aws_region, not whatever region the caller_identity resolver happened to use")
	})

	t.Run("GetCallerIdentityが失敗した場合、Loadはエラーを返す", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		client := mock.NewMockSTSClient(ctrl)
		client.EXPECT().GetCallerIdentity(gomock.Any(), gomock.Any()).Return(nil, errors.New("access denied")).Times(1)

		l := stefunny.NewConfigLoader(nil, nil)
		l.SetSTSClient(client)
		_, err := l.Load(context.Background(), "testdata/caller_identity.jsonnet")
		require.ErrorContains(t, err, "failed to get caller identity")
	})
}
