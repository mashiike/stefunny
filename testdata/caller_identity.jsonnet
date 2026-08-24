{
  required_version: '>=v0.5.0',
  state_machine: {
    name: 'Hello',
    role_arn: 'arn:aws:iam::' + std.native('caller_identity')().Account + ':role/StepFunctions-Hello-Role',
    definition: 'hello_world.asl.json',
  },
}
