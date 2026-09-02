package stefunny_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/mashiike/stefunny"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestOutputFormatter_JSON(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		data     *stefunny.ListStateMachineVersionsOutput
		expected string
	}{
		{
			name:     "nilデータはからの配列を返す",
			data:     nil,
			expected: `[]`,
		},
		{
			name: "バージョン1件をJSONとして整形する",
			data: &stefunny.ListStateMachineVersionsOutput{
				Versions: []stefunny.StateMachineVersionListItem{
					{
						Version:      1,
						Aliases:      []string{"current"},
						CreationDate: time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC),
						RevisionID:   "rev-1",
						Description:  "desc",
					},
				},
			},
			expected: `[
				{
					"StateMachineVersionArn": "",
					"version": 1,
					"aliases": ["current"],
					"description": "desc",
					"creation_date": "2021-01-02T03:04:05Z",
					"revision_id": "rev-1"
				}
			]`,
		},
		{
			name: "マルチバイト文字を含むバージョンをJSONとして整形する",
			data: &stefunny.ListStateMachineVersionsOutput{
				Versions: []stefunny.StateMachineVersionListItem{
					{
						Version:      1,
						Aliases:      []string{"現行版"},
						CreationDate: time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC),
						RevisionID:   "rev-1",
						Description:  "説明文です",
					},
				},
			},
			expected: `[
				{
					"StateMachineVersionArn": "",
					"version": 1,
					"aliases": ["現行版"],
					"description": "説明文です",
					"creation_date": "2021-01-02T03:04:05Z",
					"revision_id": "rev-1"
				}
			]`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f := stefunny.OutputFormatter{Data: c.data, Format: "json"}
			s, err := f.JSON()
			require.NoError(t, err)
			require.JSONEq(t, c.expected, s)
		})
	}
}

func TestOutputFormatter_TSV(t *testing.T) {
	t.Parallel()
	creationDate := time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC)
	cases := []struct {
		name     string
		item     stefunny.StateMachineVersionListItem
		expected string
	}{
		{
			name: "ASCIIのみのバージョン",
			item: stefunny.StateMachineVersionListItem{
				Version:      1,
				Aliases:      []string{"current", "latest"},
				CreationDate: creationDate,
				RevisionID:   "rev-1",
				Description:  "desc",
			},
			expected: "1\tcurrent,latest\t%s\trev-1\tdesc\n",
		},
		{
			name: "マルチバイト文字を含むバージョン",
			item: stefunny.StateMachineVersionListItem{
				Version:      1,
				Aliases:      []string{"現行版"},
				CreationDate: creationDate,
				RevisionID:   "rev-1",
				Description:  "説明文です",
			},
			expected: "1\t現行版\t%s\trev-1\t説明文です\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			data := &stefunny.ListStateMachineVersionsOutput{
				Versions: []stefunny.StateMachineVersionListItem{c.item},
			}
			f := stefunny.OutputFormatter{Data: data, Format: "tsv"}
			expected := fmt.Sprintf(c.expected, creationDate.Local().Format(time.RFC3339))
			require.Equal(t, expected, f.TSV())
		})
	}
}

func TestOutputFormatter_Table(t *testing.T) {
	t.Parallel()
	creationDate := time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC)
	cases := []struct {
		name        string
		versions    []stefunny.StateMachineVersionListItem
		wantContain []string
		wantRowSep  bool
	}{
		{
			name: "ASCIIのみのバージョン",
			versions: []stefunny.StateMachineVersionListItem{
				{
					Version:      1,
					Aliases:      []string{"current"},
					CreationDate: creationDate,
					RevisionID:   "rev-1",
					Description:  "desc",
				},
			},
			wantContain: []string{
				"VERSION", "ALIASES", "CREATION DATE", "REVISION ID", "DESCRIPTION",
				"1", "current", creationDate.Local().Format(time.RFC3339), "rev-1", "desc",
			},
			wantRowSep: true,
		},
		{
			name: "マルチバイト文字を含むバージョン",
			versions: []stefunny.StateMachineVersionListItem{
				{
					Version:      1,
					Aliases:      []string{"現行版"},
					CreationDate: creationDate,
					RevisionID:   "rev-1",
					Description:  "説明文です",
				},
			},
			wantContain: []string{
				"VERSION", "ALIASES", "CREATION DATE", "REVISION ID", "DESCRIPTION",
				"1", "現行版", creationDate.Local().Format(time.RFC3339), "rev-1", "説明文です",
			},
			wantRowSep: true,
		},
		{
			name:     "バージョン0件でもヘッダーのみ描画される",
			versions: nil,
			wantContain: []string{
				"VERSION", "ALIASES", "CREATION DATE", "REVISION ID", "DESCRIPTION",
			},
			wantRowSep: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			data := &stefunny.ListStateMachineVersionsOutput{Versions: c.versions}
			f := stefunny.OutputFormatter{Data: data, Format: "table"}
			s, err := f.Table()
			require.NoError(t, err)
			// CreationDateのオフセット表記はテスト実行環境のタイムゾーンで文字数が変わり罫線幅も変わるため、
			// 罫線込みの完全一致ではなく構成要素の存在確認にとどめる。
			require.Contains(t, s, "┌") // tablewriter v1系のUnicode box-drawing罫線であることの確認
			for _, want := range c.wantContain {
				require.Contains(t, s, want)
			}
			if c.wantRowSep {
				require.Contains(t, s, "├", "データ行がある場合はヘッダーとの区切り線があるはず")
			} else {
				require.NotContains(t, s, "├", "データ行が無い場合は区切り線が出ないはず")
			}
		})
	}
}

func TestOutputFormatter_Render(t *testing.T) {
	t.Parallel()
	data := &stefunny.ListStateMachineVersionsOutput{
		Versions: []stefunny.StateMachineVersionListItem{
			{Version: 1, RevisionID: "rev-1"},
		},
	}
	cases := []struct {
		format string
		want   func(t *testing.T, f stefunny.OutputFormatter) string
	}{
		{
			format: "json",
			want: func(t *testing.T, f stefunny.OutputFormatter) string {
				t.Helper()
				s, err := f.JSON()
				require.NoError(t, err)
				return s
			},
		},
		{
			format: "tsv",
			want: func(t *testing.T, f stefunny.OutputFormatter) string {
				t.Helper()
				return f.TSV()
			},
		},
		{
			format: "table",
			want: func(t *testing.T, f stefunny.OutputFormatter) string {
				t.Helper()
				s, err := f.Table()
				require.NoError(t, err)
				return s
			},
		},
		{
			format: "",
			want: func(t *testing.T, f stefunny.OutputFormatter) string {
				t.Helper()
				s, err := f.Table()
				require.NoError(t, err)
				return s
			},
		},
	}
	for _, c := range cases {
		t.Run("format="+c.format, func(t *testing.T) {
			t.Parallel()
			f := stefunny.OutputFormatter{Data: data, Format: c.format}
			s, err := f.Render()
			require.NoError(t, err)
			require.Equal(t, c.want(t, f), s)
		})
	}
}

func TestApp_Versions(t *testing.T) {
	cases := []struct {
		casename   string
		opt        stefunny.VersionsOption
		setupMocks func(t *testing.T, m *mocks)
		wantErr    bool
	}{
		{
			casename: "正常系",
			opt:      stefunny.VersionsOption{Format: "tsv"},
			setupMocks: func(t *testing.T, m *mocks) {
				sm := &stefunny.StateMachine{
					StateMachineArn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Hello"),
				}
				m.sfn.EXPECT().DescribeStateMachine(gomock.Any(), &stefunny.DescribeStateMachineInput{
					Name: "Hello",
				}).Return(sm, nil).Times(1)
				m.sfn.EXPECT().ListStateMachineVersions(gomock.Any(), sm).Return(
					&stefunny.ListStateMachineVersionsOutput{
						Versions: []stefunny.StateMachineVersionListItem{
							{Version: 1, RevisionID: "rev-1"},
						},
					},
					nil,
				).Times(1)
			},
		},
		{
			casename: "state machineが存在しない場合は何もせず終了する",
			opt:      stefunny.VersionsOption{Format: "tsv"},
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.EXPECT().DescribeStateMachine(gomock.Any(), &stefunny.DescribeStateMachineInput{
					Name: "Hello",
				}).Return(nil, stefunny.ErrStateMachineDoesNotExist).Times(1)
			},
		},
		{
			casename: "DescribeStateMachineが一般エラーを返したらエラーを返す",
			opt:      stefunny.VersionsOption{Format: "tsv"},
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.EXPECT().DescribeStateMachine(gomock.Any(), &stefunny.DescribeStateMachineInput{
					Name: "Hello",
				}).Return(nil, fmt.Errorf("some AWS error")).Times(1)
			},
			wantErr: true,
		},
		{
			casename: "PurgeStateMachineVersionsが失敗したらエラーを返す",
			opt:      stefunny.VersionsOption{Format: "tsv", Delete: true, KeepVersions: 3},
			setupMocks: func(t *testing.T, m *mocks) {
				sm := &stefunny.StateMachine{
					StateMachineArn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Hello"),
				}
				m.sfn.EXPECT().DescribeStateMachine(gomock.Any(), &stefunny.DescribeStateMachineInput{
					Name: "Hello",
				}).Return(sm, nil).Times(1)
				m.sfn.EXPECT().PurgeStateMachineVersions(gomock.Any(), sm, 3).Return(fmt.Errorf("some AWS error")).Times(1)
			},
			wantErr: true,
		},
		{
			casename: "Delete指定時はPurgeStateMachineVersionsを呼ぶ",
			opt:      stefunny.VersionsOption{Format: "tsv", Delete: true, KeepVersions: 3},
			setupMocks: func(t *testing.T, m *mocks) {
				sm := &stefunny.StateMachine{
					StateMachineArn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Hello"),
				}
				m.sfn.EXPECT().DescribeStateMachine(gomock.Any(), &stefunny.DescribeStateMachineInput{
					Name: "Hello",
				}).Return(sm, nil).Times(1)
				m.sfn.EXPECT().PurgeStateMachineVersions(gomock.Any(), sm, 3).Return(nil).Times(1)
				m.sfn.EXPECT().ListStateMachineVersions(gomock.Any(), sm).Return(
					&stefunny.ListStateMachineVersionsOutput{}, nil,
				).Times(1)
			},
		},
		{
			casename: "ListStateMachineVersionsが失敗したらエラーを返す",
			opt:      stefunny.VersionsOption{Format: "tsv"},
			setupMocks: func(t *testing.T, m *mocks) {
				sm := &stefunny.StateMachine{
					StateMachineArn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Hello"),
				}
				m.sfn.EXPECT().DescribeStateMachine(gomock.Any(), &stefunny.DescribeStateMachineInput{
					Name: "Hello",
				}).Return(sm, nil).Times(1)
				m.sfn.EXPECT().ListStateMachineVersions(gomock.Any(), sm).Return(
					nil, fmt.Errorf("some AWS error"),
				).Times(1)
			},
			wantErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			LoggerSetup(t, "debug")
			mocks := NewMocks(t)
			defer mocks.Finish()
			if c.setupMocks != nil {
				c.setupMocks(t, mocks)
			}
			app := newMockApp(t, "testdata/stefunny.yaml", mocks)
			err := app.Versions(context.Background(), c.opt)
			if c.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
