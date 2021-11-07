##　Local開発について

### testdata/terraform.tfstate の更新について

対応する `.tfファイル` は `testdata/localstack.tf` です。
以下の手順で、localstackに対してterraform applyすることで更新します。

```shell
$ make setup
$ make plan
$ make apply
$ make teardown
```

### Local環境でstefunnyを使う場合について

configには endpointsというものがあります。
configに以下のように追記することで、localstackとsfn-localにリクエストが向きます

```yaml
endpoints:
  stepfunctions: http://localhost:8083
  cloudwatchlogs: http://localhost:4566
  sts: http://localhost:4566
  eventbridge: http://localhost:4566
```
