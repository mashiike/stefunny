package stefunny

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

func (app *App) Delete(ctx context.Context, opt DeleteOption) error {
	log.Println("[info] Starting delete", opt.DryRunString())
	stateMachine, err := app.sfn.DescribeStateMachine(ctx, app.cfg.StateMachine.Name)
	if err != nil {
		return fmt.Errorf("failed to describe current state machine status: %w", err)
	}
	if stateMachine.Status == sfntypes.StateMachineStatusDeleting {
		log.Printf("[info] %s aleady deleting... %s\n", *stateMachine.StateMachineArn, opt.DryRunString())
		return nil
	}
	log.Printf("[notice] target state machine is %s (creation_date:%s) %s\n", *stateMachine.StateMachineArn, stateMachine.CreationDate, opt.DryRunString())
	if opt.DryRun {
		log.Println("[info] dry run ok")
		return nil
	}
	if !opt.Force {
		name, err := prompt(ctx, `Enter the state machine name to DELETE`, "")
		if err != nil {
			return err
		}
		if !strings.EqualFold(name, app.cfg.StateMachine.Name) {
			log.Println("[info] Aborted")
			return errors.New("confirmation failed")
		}
	}
	_, err = app.sfn.DeleteStateMachine(ctx, &sfn.DeleteStateMachineInput{
		StateMachineArn: stateMachine.StateMachineArn,
	})
	if err != nil {
		return fmt.Errorf("failed to delete state machine status: %w", err)
	}
	log.Println("[info] finish delete", opt.DryRunString())
	return nil
}
