package pubsub

import (
	"fmt"
	"job_processor/shutdown"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

const JobUpdateSubject = "job.update"
const natsServerURL = "nats://localhost:4222"
const closeSignal = "close"

func StartPubSub() (err error) {
	ns, err := server.NewServer(&server.Options{Port: 4222})
	if err != nil {
		logger.LogErr(err, "Failed to start NATS server")
		return err
	}
	go ns.Start()

	shutdown.RegisterHook(func(_ time.Duration) error {
		logger.Info("Shutting down NATS server")
		ns.Shutdown()
		return nil
	})
	return
}

// ListenForUpdates listens for updates from the job manager and publishes into Nats
func ListenForUpdates(updates <-chan any) (err error) {
	nc, err := nats.Connect(natsServerURL)
	fmt.Println("Listening for job updates...")

	go func() {
		for range updates {
			err := nc.Publish(JobUpdateSubject, []byte("updated"))
			if err != nil {
				logger.LogErr(serr.Wrap(err, "Failed to publish update"))
				return
			}
		}
	}()

	return
}

func SubscribeToUpdates(out chan any) (sub *nats.Subscription, err error) {
	nc, err := nats.Connect(natsServerURL)
	if err != nil {
		logger.LogErr(err, "Failed to connect to NATS server")
		return sub, err
	}

	sub, err = nc.Subscribe(JobUpdateSubject, func(msg *nats.Msg) {
		select {
		case out <- string(msg.Data):
			fmt.Println("Job update notification sent")
		default: // Non-blocking send to avoid blocking if no one is listening
			// If the channel is full, we don't want to block
		}
	})
	if err != nil {
		logger.LogErr(err, "Failed to subscribe to updates")
		return sub, err
	}

	logger.Info("Subscribed to job updates")

	shutdown.RegisterHook(func(_ time.Duration) error {
		logger.Info("Shutting down NATS subscription")
		sub.Unsubscribe()

		select {
		case out <- closeSignal:
			fmt.Println("Close signal sent")
		default:
			fmt.Println("Could not send close signal as output channel is full")
		}

		return nil
	})

	return
}
