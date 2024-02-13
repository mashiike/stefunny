# stefunny

![Latest GitHub release](https://img.shields.io/github/release/mashiike/stefunny.svg)
![Github Actions test](https://github.com/mashiike/stefunny/workflows/Test/badge.svg?branch=main)
[![Go Report Card](https://goreportcard.com/badge/mashiike/stefunny)](https://goreportcard.com/report/mashiike/stefunny)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/mashiike/stefunny/blob/master/LICENSE)

stefunny is a deployment tool for [AWS StepFunctions](https://aws.amazon.com/step-functions/) state machine and the accompanying [AWS EventBridge](https://aws.amazon.com/eventbridge/) rule and scheudle.

stefunny does,

- Create a state machine.
- Create a EventBridge rule and EventBridge Scheduler schedule.
- Deploy state machine/ EventBridge rule / EventBridge Scheduler schedule/ StateMachine Alias.
- Rollback to the previous version of the state machine.
- Manage state machine versions.
- Show status of the state machine.

That's all for now.

stefunny does not,

- Manage resources related to the StepFunctions state machine.
    - e.g. IAM Role, Resources called by state machine, Trigger rule that is not a schedule, CloudWatch LogGroup, etc...
- Manage StepFunctions Activities and Activity Worker.

If you hope to manage these resources **collectively**, we recommend other deployment tools ([AWS SAM](https://aws.amazon.com/serverless/sam/), [Serverless Framework](https://serverless.com/), etc.).

If you hope to manage these resources **partially individually**, we recommend the following tools:

 - [terraform](https://www.terraform.io/) for IAM Role, CloudWatch LogGroups, etc... 
 - [lambroll](https://github.com/fujiwara/lambroll) for AWS Lambda function.
 - [ecspresso](https://github.com/kayac/ecspresso) for AWS ECS Task.

## Install

### Homebrew (macOS and Linux)

```console
$ brew install mashiike/tap/stefunny
```

### aqua

[aqua](https://aquaproj.github.io/) is a declarative CLI Version Manager.

```console
$ aqua g -i mashiike/stefunny
```

### Binary packages

[Releases](https://github.com/mashiike/stefunny/releases)

### GitHub Actions

Action mashiike/stefunny@v0 installs stefunny binary for Linux into /usr/local/bin. This action runs install only.

```yml
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: mashiike/stefunny@v0
        with:
          version: v0.6.0 
      - run: |
          stefunny deploy
```
 
## QuickStart 
Try migrate your existing StepFunctions StateMachine to stefunny.

```console
$ mkdir hello
$ cd hello
$ stefunny init --state-machine Hello     
2024/02/13 16:07:53 [notice] StateMachine/Hello save config to /home/user/hello/stefunny.yaml
2024/02/13 16:07:53 [notice] StateMachine/Hello save state machine definition to /home/user/hello/definition.asl.json
```

Edit the definition.asl.json and stefunny.yaml.

Now you can deploy state machine `Hello` using `stefunny deploy`.

```console
$ stefunny deploy
2024/02/13 16:18:42 [info] Starting deploy 
2024/02/13 16:18:42 [info] update state machine `arn:aws:states:ap-northeast-1:123456789012:stateMachine:Hello:2`
2024/02/13 16:18:42 [info] update current alias `arn:aws:states:ap-northeast-1:123456789012:stateMachine:Hello:current`
2024/02/13 16:18:42 [info] deploy state machine `Hello`(at `2024-02-13 07:17:48.178 +0000 UTC`)
2024/02/13 16:18:43 [info] finish deploy 
```

## Usage

```console
Usage: stefunny <command>

stefunny is a deployment tool for AWS StepFunctions state machine

Flags:
  -h, --help                      Show context-sensitive help.
      --log-level="info"          Set log level (debug, info, notice, warn, error) ($STEFUNNY_LOG_LEVEL)
  -c, --config="stefunny.yaml"    Path to config file ($STEFUNNY_CONFIG)
      --tfstate=STRING            URL to terraform.tfstate referenced in config ($STEFUNNY_TFSTATE)
      --ext-str=,...              external string values for Jsonnet
      --ext-code=,...             external code values for Jsonnet
      --region=""                 AWS region ($AWS_REGION)
      --alias="current"           Alias name for state machine ($STEFUNNY_ALIAS)

Commands:
  version
    Show version

  init --state-machine=STRING
    Initialize stefunny configuration

  delete
    Delete state machine and schedule rules

  deploy
    Deploy state machine and schedule rules

  rollback
    Rollback state machine

  schedule --enabled --disabled
    Enable or disable schedule rules (deprecated)

  render <targets> ...
    Render state machine definition

  execute
    Execute state machine

  versions
    Manage state machine versions

  diff
    Show diff of state machine definition and trigers

  pull
    Pull state machine definition

  studio
    Show Step Functions workflow studio URL

  status
    Show status of state machine

Run "stefunny <command> --help" for more information on a command.
```

### Init 

`stepfunny init` initialize stefunny.yaml and definition file by existing state machine.

```console
Usage: stefunny init --state-machine=STRING

Initialize stefunny configuration

Flags:
  -h, --help                                Show context-sensitive help.
      --log-level="info"                    Set log level (debug, info, notice, warn, error) ($STEFUNNY_LOG_LEVEL)
  -c, --config="stefunny.yaml"              Path to config file ($STEFUNNY_CONFIG)
      --tfstate=STRING                      URL to terraform.tfstate referenced in config ($STEFUNNY_TFSTATE)
      --ext-str=,...                        external string values for Jsonnet
      --ext-code=,...                       external code values for Jsonnet
      --region=""                           AWS region ($AWS_REGION)
      --alias="current"                     Alias name for state machine ($STEFUNNY_ALIAS)

      --state-machine=STRING                AWS StepFunctions state machine name ($STATE_MACHINE_NAME)
  -d, --definition="definition.asl.json"    Path to state machine definition file ($DEFINITION_FILE_PATH)
      --env=ENV,...                         templateize environment variables
      --must-env=MUST-ENV,...               templateize must environment variables
      --skip-trigger                        Skip trigger
```

created file foramt are checked file extension. `.json` saved as json, `.jsonnet` saved as jsonnet, `.yaml` or `.yml` saved as yaml.

If you manage the aws resources by terraform, you can use `--tfstate` flag with `stefunny init` command.

```console
$ export ENV=dev
$ stefunny init --state-machine dev-Hello --tfstate s3://my-bucket/terraform.tfstate --must-env ENV
```
in this case, saved config and definition file are templatized by `text/template` 

for example, the saved config file is like this.

```yaml
aws_region: ap-northeast-1
required_version: ">=v0.6.0"
state_machine:
  definition: definition.asl.json
  logging_configuration:
    destinations:
    - cloudwatch_logs_log_group:
        log_group_arn: "{{ tfstate `aws_cloudwatch_log_group.state_machine.arn` }}:*"
    include_execution_data: true
    level: ALL
  name: "{{ must_env `ENV` }}-Hello"
  role_arn: "{{ tfstate `aws_iam_role.state_machine.arn` }}"
  tags:
  - key: Name
    value: "{{ must_env `ENV` }}-Hello"
  tracing_configuration:
    enabled: true
  type: STANDARD
```

### Deploy

```console
Usage: stefunny deploy

Deploy state machine and schedule rules

Flags:
  -h, --help                          Show context-sensitive help.
      --log-level="info"              Set log level (debug, info, notice, warn, error) ($STEFUNNY_LOG_LEVEL)
  -c, --config="stefunny.yaml"        Path to config file ($STEFUNNY_CONFIG)
      --tfstate=STRING                URL to terraform.tfstate referenced in config ($STEFUNNY_TFSTATE)
      --ext-str=,...                  external string values for Jsonnet
      --ext-code=,...                 external code values for Jsonnet
      --region=""                     AWS region ($AWS_REGION)
      --alias="current"               Alias name for state machine ($STEFUNNY_ALIAS)

      --dry-run                       Dry run
      --skip-state-machine            Skip deploy state machine
      --skip-trigger                  Skip deploy trigger
      --version-description=STRING    Version description
      --keep-versions=0               Number of latest versions to keep. Older versions will be deleted. (Optional value: default 0)
      --trigger-enabled               Enable trigger
      --trigger-disabled              Disable trigger
      --[no-]unified                  when dry run, output unified diff
```
stefunny deploy works as below.

- Create / Update State Machine from config file and definition file(yaml/json/jsonnet)
  - Replace {{ env `FOO` `bar` }} syntax in the config file and definition file to environment variable "FOO".
    If "FOO" is not defined, replaced by "bar"
  - Replace {{ must_env `FOO` }} syntax in the config file and definition file to environment variable "FOO".
    If "FOO" is not defined, abort immediately.
  - If a terraform state is given in config, replace the {{tfstate `<tf resource name>`}} syntax in the config file and definition file with reference to the state content.
- Publish new version of the state machine.
- Update the alias to the new version.
- Create/ Update EventBridge rule.
- Create/ Update EventBridge Scheduler schedule.

### Rollback 

```console
Usage: stefunny rollback

Rollback state machine

Flags:
  -h, --help                      Show context-sensitive help.
      --log-level="info"          Set log level (debug, info, notice, warn, error) ($STEFUNNY_LOG_LEVEL)
  -c, --config="stefunny.yaml"    Path to config file ($STEFUNNY_CONFIG)
      --tfstate=STRING            URL to terraform.tfstate referenced in config ($STEFUNNY_TFSTATE)
      --ext-str=,...              external string values for Jsonnet
      --ext-code=,...             external code values for Jsonnet
      --region=""                 AWS region ($AWS_REGION)
      --alias="current"           Alias name for state machine ($STEFUNNY_ALIAS)

      --dry-run                   Dry run
      --keep-version              Keep current version, no delete
```

`stefunny deploy` create/update alias `current` to the published state machine version on deploy.

`stefunny rollback` works as below.

1. Find previous one version of state machine.
2. Update alias `current` to the previous version.
3. default delete old version of state machine. (when `--keep-version` specified, not delete old version of state machine)

### Studio and Pull 

If you use AWS Step Functions Workflow Studio, you can open the studio URL with `stefunny studio` command.

```console
Usage: stefunny studio

Show Step Functions workflow studio URL

Flags:
  -h, --help                      Show context-sensitive help.
      --log-level="info"          Set log level (debug, info, notice, warn, error) ($STEFUNNY_LOG_LEVEL)
  -c, --config="stefunny.yaml"    Path to config file ($STEFUNNY_CONFIG)
      --tfstate=STRING            URL to terraform.tfstate referenced in config ($STEFUNNY_TFSTATE)
      --ext-str=,...              external string values for Jsonnet
      --ext-code=,...             external code values for Jsonnet
      --region=""                 AWS region ($AWS_REGION)
      --alias="current"           Alias name for state machine ($STEFUNNY_ALIAS)

      --open                      open workflow studio
```

`stefunny studio` command shows the studio URL. If `--open` flag is specified, open the studio URL in the browser.
Edit state machine on Workflow Studio, and pull the definition file with `stefunny pull` command.

```console
Usage: stefunny pull

Pull state machine definition

Flags:
  -h, --help                      Show context-sensitive help.
      --log-level="info"          Set log level (debug, info, notice, warn, error) ($STEFUNNY_LOG_LEVEL)
  -c, --config="stefunny.yaml"    Path to config file ($STEFUNNY_CONFIG)
      --tfstate=STRING            URL to terraform.tfstate referenced in config ($STEFUNNY_TFSTATE)
      --ext-str=,...              external string values for Jsonnet
      --ext-code=,...             external code values for Jsonnet
      --region=""                 AWS region ($AWS_REGION)
      --alias="current"           Alias name for state machine ($STEFUNNY_ALIAS)

      --[no-]templateize          templateize output
      --qualifier=STRING          qualifier for the version
```

`stefunny pull` command pull the definition file from the state machine and save it to the file.

### config file (yaml)

```yaml
required_version: ">=v0.6.0"
aws_region: "{{ env `AWS_REGION` `ap-northeast-1` }}"

state_machine:
  name: "{{ must_env `ENV` }}-Hello"
  definition: definition.asl.json
  logging_configuration:
    destinations:
      - cloudwatch_logs_log_group:
          log_group_arn: "{{ tfstate `aws_cloudwatch_log_group.state_machine.arn` }}:*"
    include_execution_data: true
    level: ALL

  role_arn: "{{ tfstate `aws_iam_role.state_machine.arn` }}"
  tracing_configuration:
    enabled: true
  type: STANDARD

tfstate:
  - location: s3://my-tfstate-bucket/terraform.tfstate

trigger:
  schedule:
    - name: "{{ must_env `ENV` }}-stefunny-test"
      group_name: default
      action_after_completion: DELETE
      flexible_time_window:
        maximum_window_in_minutes: 240.0
        mode: FLEXIBLE
      schedule_expression: at(2024-02-29T00:01:00)
      target:
        retry_policy:
          maximum_event_age_in_seconds: 86400.0
          maximum_retry_attempts: 185.0
        role_arn: "{{ tfstate `aws_iam_role.event_bridge_scheduler.arn` }}"

  event:
    - name: "{{ must_env `ENV` }}-stefunny-test"
      event_bus_name: default
      event_pattern: "{{ file `event_pattern.json` | json_escape }}"
      role_arn: "{{ tfstate `aws_iam_role.event_bridge.arn` }}"

```

Configuration files and definition files are read with `text/template`, stefunny has template functions env, must_env, file, json_escape and tfstate.


### Template syntax

stefunny uses the [text/template standard package in Go](https://pkg.go.dev/text/template) to render template files, and parses as YAML/JSON/Jsonnet. 

#### `env`

```
"{{ env `NAME` `default value` }}"
```

If the environment variable `NAME` is set, it will replace with its value. If it's not set, it will replace with "default value".

#### `must_env`

```
"{{ must_env `NAME` }}"
```

It replaces with the value of the environment variable `NAME`. If the variable isn't set at the time of execution, stefunny will panic and stop forcefully.

By defining values that can cause issues when running without meaningful values with must_env, you can prevent unintended deployments.

#### `json_escape`

```
"{{ must_env `JSON_VALUE` | json_escape }}"
```

It escapes values as JSON strings. Use it when you want to escape values that need to be embedded as strings and require escaping, like quotes.

#### `file`

```
"{{ file `path/to/file` }}"
```

#### `tfstate`

If written `tfstate` section in the configuration file, it will be use `tfstate` template function. as following.

stefunny.yaml
```yaml
required_version: ">v0.0.0"

state_machine:
  name: send_sns
  definition: send_sns.asl.jsonnet
  role_arn: "{{ tfstae `aws_iam_role.stepfunctions.arn` }}"
  logging_configuration:
    level: OFF

tfstate:
  - path: "./terraform.tfstate"
  - url: s3://my-bucket/terraform.tfstate
    func_prefix: s3_
```

send_sns.asl.jsonnet
```jsonnet
{
  Comment: "A simple AWS Step Functions state machine that sends a message to an SNS topic", 
  StartAt: "Send SNS Message",
  States: {
    "Send SNS Message": {
      Type: "Task",
      Resource: "arn:aws:states:::sns:publish",
      Parameters: {
        "TopicArn": "{{ s3_tfstate `aws_sns_topic.topic.arn` }}",
        "Message.$": "$"
      },
      End: true,
    }
  }
}
```

`{{ tfstate "resource_type.resource_name.attr" }}` will expand to an attribute value of the resource in tfstate.

`{{ tfstatef "resource_type.resource_name['%s'].attr" "index" }}` is similar to `{{ tfstatef "resource_type.resource_name['index'].attr" }}`. This function is useful to build a resource address with environment variables.

```
{{ tfstatef `aws_subnet.ecs['%s'].id` (must_env `SERVICE`) }}
```

This function uses [tfstate-lookup](https://github.com/fujiwara/tfstate-lookup) to load tfstate.


## Special Thanks

@fujiwara has given me naming idea of stefunny.

##  Inspire tools

 - [lambroll](https://github.com/fujiwara/lambroll)
 - [ecspresso](https://github.com/kayac/ecspresso)
## LICENSE

MIT License

Copyright (c) 2021 IKEDA Masashi
