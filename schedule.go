package stefunny

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/scheduler"
)

type Schedule struct {
	scheduler.CreateScheduleInput
	Arn          *string    `min:"1" type:"string"`
	CreationDate *time.Time `type:"timestamp"`
}
