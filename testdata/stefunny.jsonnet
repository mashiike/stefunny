{
  required_version: '>=v0.5.0',
  state_machine: {
    name: 'Hello',
    role_arn: 'arn:aws:iam::123456789012:role/StepFunctions-Hello-Role',
    definition: 'hello_world.asl.json',
    logging: {
      level: 'ALL',
      destination: {
        log_group: '/steps/hello',
      },
    },
  },
}
