# stefunny

![Latest GitHub release](https://img.shields.io/github/release/mashiike/stefunny.svg)
![Github Actions test](https://github.com/mashiike/stefunny/workflows/Test/badge.svg?branch=main)
[![Go Report Card](https://goreportcard.com/badge/mashiike/stefunny)](https://goreportcard.com/report/mashiike/stefunny)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/mashiike/stefunny/blob/master/LICENSE)

stefunny is a deployment tool for [AWS StepFunctions](https://aws.amazon.com/step-functions/) state machine and the accompanying [AWS EventBrdige](https://aws.amazon.com/eventbridge/) rule.

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
   stefunny - A command line tool for deployment StepFunctions and EventBrdige

USAGE:
   stefunny [global options] command [command options] [arguments...]

COMMANDS:
   create    create StepFunctions StateMachine.
   delete    delete StepFunctions StateMachine.
   deploy    deploy StepFunctions StateMachine and Event Bridge Rule.
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
  logging:
    level: ALL
    destination:
      log_group:  "{{ must_env `LOG_GROUP` }}"

tags:
  env: "{{ must_env `ENV` }}" 

schedule:
  expression: rate(1 hour)
  role_arn: "{{ tfstae `aws_iam_role.eventbridge.arn` }}"
```

Configuration files and definition files are read by go-config. go-config has template functions env, must_env and json_escape.

## Special Thanks

@fujiwara has given me naming idea of stefunny.

##  Inspire tools

 - [lambroll](https://github.com/fujiwara/lambroll)
 - [ecspresso](https://github.com/kayac/ecspresso)
## LICENSE

MIT License

Copyright (c) 2021 IKEDA Masashi
