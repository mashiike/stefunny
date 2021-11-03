{
  Type: "Task",
  Resource: "arn:aws:states:::aws-sdk:s3:listObjects",
  parameters(bucket_name):: {
      Bucket: bucket_name
  },
}
