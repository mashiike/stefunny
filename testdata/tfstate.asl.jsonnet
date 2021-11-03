local pass = import 'libs/pass.libsonnet';
local list_objects = import 'libs/list_objects.libsonnet';

{
  Comment: "A Hello World example of the Amazon States Language using Pass states",
  StartAt: "Hello",
  States: {
    Hello: pass+{
      Next: "World"
    },
    World: list_objects+{
      Parameters: self.parameters("{{ tfstate `aws_s3_bucket.hoge.bucket` }}"),
      End: true
    }
  }
}
