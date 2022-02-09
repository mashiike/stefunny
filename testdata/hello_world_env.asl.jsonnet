local pass = import 'libs/pass.libsonnet';

{
  Comment: "A Hello World example of the Amazon States Language using Pass states",
  StartAt: "{{ must_env `START_AT` }}",
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
