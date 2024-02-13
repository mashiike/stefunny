# stefunny

![Latest GitHub release](https://img.shields.io/github/release/mashiike/stefunny.svg)
![Github Actions test](https://github.com/mashiike/stefunny/workflows/Test/badge.svg?branch=main)
[![Go Report Card](https://goreportcard.com/badge/mashiike/stefunny)](https://goreportcard.com/report/mashiike/stefunny)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/mashiike/stefunny/blob/master/LICENSE)

stefunny is a deployment tool for [AWS StepFunctions](https://aws.amazon.com/step-functions/) state machine and the accompanying [AWS EventBridge](https://aws.amazon.com/eventbridge/) rule.

stefunny does,

- Create a state machine.
- Create a scheduled rule.
- Update state machine definition / configuration / tags 
- Update a scheduled rule.

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

### Binary packages

[Releases](https://github.com/mashiike/stefunny/releases)

## Usage

```console
NAME:
   stefunny - A command line tool for deployment StepFunctions and EventBridge

USAGE:
   stefunny [global options] command [command options] [arguments...]

COMMANDS:
   create    create StepFunctions StateMachine.
   delete    delete StepFunctions StateMachine.
   deploy    deploy StepFunctions StateMachine and Event Bridge Rule.
   execute   execute state machine
   init      Initialize stefunny from an existing StateMachine
   render    render state machine definition(the Amazon States Language) as a dot file
   schedule  schedule Bridge Rule without deploy StepFunctions StateMachine.
   version   show version info.
   help, h   Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --config FILE, -c FILE  Load configuration from FILE (default: config.yaml) [$STEFUNNY_CONFIG]
   --log-level value       Set log level (debug, info, notice, warn, error) (default: info) [$STEFUNNY_LOG_LEVEL]
   --tfstate value         URL to terraform.tfstate referenced in config [$STEFUNNY_TFSTATE]
   --help, -h              show help (default: false)
```

## Quick Start

stefunny can easily manage for your existing StepFunctions StateMachine by codes.

Try `stefunny init` for your StepFunctions StateMachine with option `--state-machine`.

```console
stefunny init --region ap-northeast-1 --config config.yaml --state-machine HelloWorld 
2021/11/23 18:08:00 [notice] StateMachine/HelloWorld save state machine definition to definition.jsonnet
2021/11/23 18:08:00 [notice] StateMachine/HelloWorld save config to config.yaml
```
**If you want to manage StateMachine definition in other formats**, use the `--definition` option and specify the definition file. The default is jsonnet format, but you can use json format (.json) and yaml format (.yaml, .yml)

Let me see the generated files config.yaml, and definition.jsonnet.

And then, you already can deploy the service by stefunny!

```console
$ stefunny deploy --config config.yaml
```

### Deploy

```console
NAME:
   stefunny deploy - deploy StepFunctions StateMachine and Event Bridge Rule.

USAGE:
   stefunny deploy [command options] [arguments...]

OPTIONS:
   --dry-run  dry run (default: false)
```
stefunny deploy works as below.

- Create / Update State Machine from config file and definition file(yaml/json/jsonnet)
  - Replace {{ env `FOO` `bar` }} syntax in the config file and definition file to environment variable "FOO".
    If "FOO" is not defined, replaced by "bar"
  - Replace {{ must_env `FOO` }} syntax in the config file and definition file to environment variable "FOO".
    If "FOO" is not defined, abort immediately.
  - If a terraform state is given in --tfstate, replace the {{tfstate `<tf resource name>`}} syntax in the config file and definition file with reference to the state content.
- Create/ Update EventBridge rule.

### Schedule Enabled/Disabled

```console
NAME:
   stefunny schedule - schedule Bridge Rule without deploy StepFunctions StateMachine.

USAGE:
   stefunny schedule [command options] [arguments...]

OPTIONS:
   --dry-run   dry run (default: false)
   --enabled   set schedule rule enabled (default: false)
   --disabled  set schedule rule disabled (default: false)
```

```console
$ stefunny -config config.yaml schedule --disabled
```

Update the rules in EventBridge to disable the state. This is done without updating the state machine.

### Render 

```console
NAME:
   stefunny render - render state machine definition(the Amazon States Language) as a dot file

USAGE:
   stefunny render [arguments...]
```

```console
$ stefunny -config config.yaml render hello_world.dot
```

The Render command reads the definition file, interprets the ASL and renders the State relationship into a DOT file.

### config file (yaml)

```yaml
required_version: ">v0.0.0"

state_machine:
  name: hello_world
  definition: hello_world.asl.jsonnet
  role_arn: "{{ tfstae `aws_iam_role.stepfunctions.arn` }}"
  logging_configuration:
    level: ALL
    destinations:
      - cloudwatch_log_group:
          log_group_arn: "{{ must_env `LOG_GROUP_Arn` }}"

tags:
  env: "{{ must_env `ENV` }}" 

tfstate:
  - path: "./terraform.tfstate"

schedule:
  expression: rate(1 hour)
  role_arn: "{{ tfstae `aws_iam_role.eventbridge.arn` }}"
```

Configuration files and definition files are read with `text/template`, stefunny has template functions env, must_env, json_escape and tfstate.


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
