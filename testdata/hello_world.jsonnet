local pass = import 'libs/pass.libsonnet';

{
  Comment: "A Hello World example of the Amazon States Language using Pass states",
  StartAt: "Hello",
  States: {
    Hello: pass+{
      Next: "World"
    },
    World: pass+{
      Result: "World",
      End: true
    }
  }
}
