# stefunny

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

 - [terarform](https://www.terraform.io/) for IAM Role, CloudWatch LogGroups, etc... 
 - [lambroll](https://github.com/fujiwara/lambroll) for AWS Lambda function.
 - [ecspresso](https://github.com/kayac/ecspresso) for AWS ECS Task.
