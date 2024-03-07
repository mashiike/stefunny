# Changelog

## [v0.7.1](https://github.com/mashiike/stefunny/compare/v0.7.0...v0.7.1) - 2024-03-07
- event bridge put rule RoleARN is duplicated info by @mashiike in https://github.com/mashiike/stefunny/pull/208

## [v0.7.0](https://github.com/mashiike/stefunny/compare/v0.6.0...v0.7.0) - 2024-03-07
- template rendring with env, tfstate section by @mashiike in https://github.com/mashiike/stefunny/pull/199
- Bump github.com/aws/aws-sdk-go-v2/service/scheduler from 1.6.6 to 1.7.0 by @dependabot in https://github.com/mashiike/stefunny/pull/198
- Bump github.com/aws/smithy-go from 1.19.0 to 1.20.0 by @dependabot in https://github.com/mashiike/stefunny/pull/194
- Bump github.com/aws/aws-sdk-go-v2/service/sfn from 1.24.7 to 1.26.1 by @dependabot in https://github.com/mashiike/stefunny/pull/201
- Bump github.com/aws/aws-sdk-go-v2/service/eventbridge from 1.28.1 to 1.30.1 by @dependabot in https://github.com/mashiike/stefunny/pull/203
- Bump github.com/aws/aws-sdk-go-v2 from 1.25.0 to 1.25.2 by @dependabot in https://github.com/mashiike/stefunny/pull/204
- Bump github.com/aws/aws-sdk-go-v2/config from 1.26.6 to 1.27.6 by @dependabot in https://github.com/mashiike/stefunny/pull/207

## [v0.6.0](https://github.com/mashiike/stefunny/compare/v0.5.0...v0.6.0) - 2024-02-13
- Feature/fix actionlit by @mashiike in https://github.com/mashiike/stefunny/pull/176
- change to github.com/alecthomas/kong by @mashiike in https://github.com/mashiike/stefunny/pull/178
- [Breaking Changes] Configure `state_machine` structure by @mashiike in https://github.com/mashiike/stefunny/pull/179
- [Bracking Changes] Remake Render command, and allow jsonnet config by @mashiike in https://github.com/mashiike/stefunny/pull/180
- Feature/refactor aws service by @mashiike in https://github.com/mashiike/stefunny/pull/181
- v0.6.0 is bridge version for configration format. output warning message by @mashiike in https://github.com/mashiike/stefunny/pull/182
- Feature: On Deploy, publish version and create/update alias. implement Rollback by @mashiike in https://github.com/mashiike/stefunny/pull/183
- refactor aws code by @mashiike in https://github.com/mashiike/stefunny/pull/184
- implement EventBridge Sceduler by @mashiike in https://github.com/mashiike/stefunny/pull/185
- add `diff` sub command by @mashiike in https://github.com/mashiike/stefunny/pull/186
- new subcommands `studio` and `pull` by @mashiike in https://github.com/mashiike/stefunny/pull/187
- Bump github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs from 1.31.0 to 1.32.0 by @dependabot in https://github.com/mashiike/stefunny/pull/189
- ARN -> Arn by @mashiike in https://github.com/mashiike/stefunny/pull/190
- Yoshina default config names by @mashiike in https://github.com/mashiike/stefunny/pull/191
- Bump golang.org/x/term from 0.16.0 to 0.17.0 by @dependabot in https://github.com/mashiike/stefunny/pull/188
- new subcommand `status`  by @mashiike in https://github.com/mashiike/stefunny/pull/192
- misc for v0.6.0 Release by @mashiike in https://github.com/mashiike/stefunny/pull/193

## [v0.5.0](https://github.com/mashiike/stefunny/compare/v0.4.3...v0.5.0) - 2024-02-01
- refactoring, Render DOT and YAML is deprecated, create Command deprecated by @mashiike in https://github.com/mashiike/stefunny/pull/167
- delete release actions by @mashiike in https://github.com/mashiike/stefunny/pull/173
- Bump actions/setup-go from 1 to 5 by @dependabot in https://github.com/mashiike/stefunny/pull/170
- update go modules by @mashiike in https://github.com/mashiike/stefunny/pull/174
- Feature/actions by @mashiike in https://github.com/mashiike/stefunny/pull/175

## [v0.4.3](https://github.com/mashiike/stefunny/compare/v0.4.2...v0.4.3) - 2022-05-18
- Compensate .PHONY in the Makefile by @ebi-yade in https://github.com/mashiike/stefunny/pull/9
- It seems that the event pattern rule is panicked when it exists, so it is collected. by @mashiike in https://github.com/mashiike/stefunny/pull/11

## [v0.4.2](https://github.com/mashiike/stefunny/compare/v0.4.1...v0.4.2) - 2022-02-15
- fix no configured schedule rule by @mashiike in https://github.com/mashiike/stefunny/pull/8

## [v0.4.1](https://github.com/mashiike/stefunny/compare/v0.4.0...v0.4.1) - 2022-02-15

## [v0.4.0](https://github.com/mashiike/stefunny/compare/v0.3.3...v0.4.0) - 2022-02-10
- Feature/ext var  ext code by @mashiike in https://github.com/mashiike/stefunny/pull/6
- implement express execute by @mashiike in https://github.com/mashiike/stefunny/pull/7

## [v0.3.3](https://github.com/mashiike/stefunny/compare/v0.3.2...v0.3.3) - 2022-02-10

## [v0.3.2](https://github.com/mashiike/stefunny/compare/v0.3.1...v0.3.2) - 2022-02-10

## [v0.3.1](https://github.com/mashiike/stefunny/compare/v0.3.0...v0.3.1) - 2022-02-10

## [v0.3.0](https://github.com/mashiike/stefunny/compare/v0.2.0...v0.3.0) - 2022-02-10
- Feature/execute cmd by @mashiike in https://github.com/mashiike/stefunny/pull/5

## [v0.2.0](https://github.com/mashiike/stefunny/compare/v0.1.1...v0.2.0) - 2022-02-09
- Feature/render format by @mashiike in https://github.com/mashiike/stefunny/pull/4

## [v0.1.1](https://github.com/mashiike/stefunny/compare/v0.1.0...v0.1.1) - 2022-01-31
- Feature/fix logging level config not working by @mashiike in https://github.com/mashiike/stefunny/pull/3

## [v0.1.0](https://github.com/mashiike/stefunny/compare/v0.0.0...v0.1.0) - 2021-11-23
- Feature/init command by @mashiike in https://github.com/mashiike/stefunny/pull/1
- Managing Schedule using ManagedBy tag by @mashiike in https://github.com/mashiike/stefunny/pull/2

## [v0.0.0](https://github.com/mashiike/stefunny/commits/v0.0.0) - 2021-11-08
