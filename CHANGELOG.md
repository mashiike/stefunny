# Changelog

## [v0.9.3](https://github.com/mashiike/stefunny/compare/v0.9.2...v0.9.3) - 2025-02-22
- Bump github.com/aws/aws-sdk-go-v2 from 1.26.1 to 1.30.3 by @dependabot in https://github.com/mashiike/stefunny/pull/287
- Bump github.com/hashicorp/go-version from 1.6.0 to 1.7.0 by @dependabot in https://github.com/mashiike/stefunny/pull/270
- Feature/update go 1.24 and change mock library by @mashiike in https://github.com/mashiike/stefunny/pull/297
- Bump goreleaser/goreleaser-action from 5 to 6 by @dependabot in https://github.com/mashiike/stefunny/pull/276

## [v0.9.2](https://github.com/mashiike/stefunny/compare/v0.9.1...v0.9.2) - 2024-05-07
- Fix: Schedule deploy. ScheduleExpressionTimezone and KeepState by @mashiike in https://github.com/mashiike/stefunny/pull/263

## [v0.9.1](https://github.com/mashiike/stefunny/compare/v0.9.0...v0.9.1) - 2024-05-07
- fix not default group. can not get exists schedule by @mashiike in https://github.com/mashiike/stefunny/pull/261

## [v0.9.0](https://github.com/mashiike/stefunny/compare/v0.8.4...v0.9.0) - 2024-05-07
- fix: Detect Related Schedules in not default group by @mashiike in https://github.com/mashiike/stefunny/pull/253
- change dependabot config: grouping update setting for aws sdk by @mashiike in https://github.com/mashiike/stefunny/pull/255
- remake: tagpr and release workflow by @mashiike in https://github.com/mashiike/stefunny/pull/256
- Bump golang.org/x/term from 0.17.0 to 0.20.0 by @dependabot in https://github.com/mashiike/stefunny/pull/252
- Bump github.com/aws/aws-sdk-go-v2 from 1.25.2 to 1.26.1 by @dependabot in https://github.com/mashiike/stefunny/pull/248
- Bump golang.org/x/net from 0.20.0 to 0.23.0 by @dependabot in https://github.com/mashiike/stefunny/pull/251
- Bump google.golang.org/protobuf from 1.32.0 to 1.33.0 by @dependabot in https://github.com/mashiike/stefunny/pull/240
- Bump the aws-sdk-go-v2 group with 6 updates by @dependabot in https://github.com/mashiike/stefunny/pull/257
- update .golangci.yaml by @mashiike in https://github.com/mashiike/stefunny/pull/259
- Bump github.com/stretchr/testify from 1.8.4 to 1.9.0 by @dependabot in https://github.com/mashiike/stefunny/pull/235
- Bump github.com/alecthomas/kong from 0.8.1 to 0.9.0 by @dependabot in https://github.com/mashiike/stefunny/pull/258
- actions-misspell: exclude auto generate by @mashiike in https://github.com/mashiike/stefunny/pull/260

## [v0.8.4](https://github.com/mashiike/stefunny/compare/v0.8.3...v0.8.4) - 2024-03-08
- more explain, diff source in config file by @mashiike in https://github.com/mashiike/stefunny/pull/232
- supression of diff eventbridge_rules by @mashiike in https://github.com/mashiike/stefunny/pull/234

## [v0.8.3](https://github.com/mashiike/stefunny/compare/v0.8.2...v0.8.3) - 2024-03-08
- no deployed stats not error by @mashiike in https://github.com/mashiike/stefunny/pull/229
- fix panic: care no version arn by @mashiike in https://github.com/mashiike/stefunny/pull/231

## [v0.8.2](https://github.com/mashiike/stefunny/compare/v0.8.1...v0.8.2) - 2024-03-08
- json escape target is not escape HTML by @mashiike in https://github.com/mashiike/stefunny/pull/227

## [v0.8.1](https://github.com/mashiike/stefunny/compare/v0.8.0...v0.8.1) - 2024-03-08
- fix if tfstate location is not local path by @mashiike in https://github.com/mashiike/stefunny/pull/225

## [v0.8.0](https://github.com/mashiike/stefunny/compare/v0.7.4...v0.8.0) - 2024-03-08
- suppression TFState Lookup Listup warn by @mashiike in https://github.com/mashiike/stefunny/pull/221
- Feature/create file as mkdir p by @mashiike in https://github.com/mashiike/stefunny/pull/223
- Add `template_file` template function by @mashiike in https://github.com/mashiike/stefunny/pull/224

## [v0.7.4](https://github.com/mashiike/stefunny/compare/v0.7.3...v0.7.4) - 2024-03-07
- state machine version no have tags by @mashiike in https://github.com/mashiike/stefunny/pull/217
- if diff set --qualifier but current not exits latest exists, get arn … by @mashiike in https://github.com/mashiike/stefunny/pull/219
- if empty string diff, no warn by @mashiike in https://github.com/mashiike/stefunny/pull/220

## [v0.7.3](https://github.com/mashiike/stefunny/compare/v0.7.2...v0.7.3) - 2024-03-07
- no diff managed by tags by @mashiike in https://github.com/mashiike/stefunny/pull/214
- if alias not exists return ErrStateMachineDooesNotExists by @mashiike in https://github.com/mashiike/stefunny/pull/216

## [v0.7.2](https://github.com/mashiike/stefunny/compare/v0.7.1...v0.7.2) - 2024-03-07
- fix test by @mashiike in https://github.com/mashiike/stefunny/pull/211
- Feature/fix env template func by @mashiike in https://github.com/mashiike/stefunny/pull/213

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
