package stefunny

import "context"

type StatusOption struct {
}

func (app *App) Status(ctx context.Context, opt StatusOption) error {
	return nil
}
