{
  required_version: '>=v0.5.0',
  state_machine: {
    name: 'Hello',
    role_arn: 'arn:aws:iam::123456789012:role/StepFunctions-Hello-Role',
    definition: 'hello_world.asl.json',
    logging_configuration: {
      level: 'ALL',
      destinations: [
        {
          cloud_watch_logs_log_group: {
            log_group_arn: 'arn:aws:logs:us-east-1:123456789012:log-group:/aws/stepfunctions/Hello',
          },
        },
      ],
    },
  },
}
