{
  Comment: 'A Hello World example of the Amazon States Language using Pass states',
  StartAt: 'Hello',
  States: {
    Hello: {
      Type: 'Pass',
      Next: 'World',
    },
    World: {
      Type: 'Pass',
      Result: 'World',
      End: true,
    },
  },
}
