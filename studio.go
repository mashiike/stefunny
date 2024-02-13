package stefunny

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
)

type StudioOption struct {
	Open bool `name:"open" help:"open workflow studio"`
}

func (app *App) Studio(ctx context.Context, opt StudioOption) error {
	awsRegion := app.cfg.AWSRegion
	if awsRegion == "" {
		if region := os.Getenv("AWS_REGION"); region != "" {
			awsRegion = region
		} else if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
			awsRegion = region
		} else {
			return fmt.Errorf("AWS region is not set")
		}
	}
	stateMachineArn, err := app.sfnSvc.GetStateMachineArn(ctx, &GetStateMachineArnInput{
		Name: app.cfg.StateMachineName(),
	})
	if err != nil {
		return fmt.Errorf("failed to get state machine arn: %w", err)
	}
	u := &url.URL{
		Scheme:   "https",
		Host:     fmt.Sprintf("%s.console.aws.amazon.com", awsRegion),
		Path:     "/states/home",
		Fragment: fmt.Sprintf("/v2/statemachines/edit/%s", stateMachineArn),
	}
	query := url.Values{
		"region": []string{awsRegion},
	}
	u.RawQuery = query.Encode()
	fmt.Println(u.String())
	if !opt.Open {
		return nil
	}
	if err := openBrowser(u.String()); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}
	return nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		return errors.New("unsupported platform")
	}
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
