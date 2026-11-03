package machinery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	ceruviaConfig "github.com/Ceruvia/grader/internal/config"
	"github.com/RichardKnop/machinery/v2/brokers/iface"
	"github.com/RichardKnop/machinery/v2/common"
	"github.com/RichardKnop/machinery/v2/config"
	"github.com/RichardKnop/machinery/v2/tasks"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// maxJobDuration ends lease extension. A job still running after it is
	// redelivered to another grader.
	maxJobDuration = time.Hour
	// maxAckDeadline is Pub/Sub's longest ack deadline. Until the grader's own
	// lease is confirmed, the message may stay leased that long.
	maxAckDeadline = 600 * time.Second
)

var (
	// lease is the ack deadline a running job keeps, extended every lease/3.
	// A grader that dies stops extending it, so its jobs reach another grader
	// within lease.
	lease = 30 * time.Second
	// rpcTimeout bounds each ack, nack or extension call, retries included.
	rpcTimeout = 10 * time.Second
	// settleRetry is the wait between failed ack or nack attempts.
	settleRetry = time.Second
	// shutdownGrace is how long a stopping grader waits for running jobs before
	// it releases their messages for redelivery. It stays below Docker's
	// default 10 s stop timeout so the release happens before the kill.
	shutdownGrace = 5 * time.Second
	// pubsubClientOptions are extra client options. Tests inject faults here.
	pubsubClientOptions []option.ClientOption
)

// pubsubBroker consumes Machinery signatures from a Pub/Sub subscription.
// Machinery's own gcppubsub broker cannot cap outstanding messages and acks
// after a Nack. Streaming pull caps only on the client and the server may send
// more, which a busy grader would then hold while others idle. So each worker
// slot pulls one message only when it is free. A message is acked once the
// Backend gave its final answer to the callback, and nacked otherwise.
type pubsubBroker struct {
	common.Broker
	client       *pubsub.Client
	subscription string
	topic        string
	unanswered   *sync.Map
	stopped      chan struct{}
	release      chan struct{}
}

func newPubSubBroker(cnf *config.Config, mq *ceruviaConfig.MessageQueueConfig, unanswered *sync.Map) (*pubsubBroker, error) {
	client, err := pubsub.NewClient(context.Background(), mq.ProjectID, pubsubClientOptions...)
	if err != nil {
		return nil, fmt.Errorf("create Pub/Sub client: %w", err)
	}
	return &pubsubBroker{
		Broker:       common.NewBroker(cnf),
		client:       client,
		subscription: fmt.Sprintf("projects/%s/subscriptions/%s", mq.ProjectID, mq.SubscriptionName),
		topic:        mq.TopicName,
		unanswered:   unanswered,
		stopped:      make(chan struct{}),
		release:      make(chan struct{}),
	}, nil
}

// StartConsuming runs concurrency worker slots until StopConsuming, or until a
// pull fails after the client's own retries. Machinery then calls it again.
func (b *pubsubBroker) StartConsuming(consumerTag string, concurrency int, p iface.TaskProcessor) (bool, error) {
	b.Broker.StartConsuming(consumerTag, concurrency, p)

	ctx, stopPulling := context.WithCancel(context.Background())
	defer stopPulling()
	go func() {
		select {
		case <-b.GetStopChan():
			stopPulling()
		case <-ctx.Done():
		}
	}()

	var wg sync.WaitGroup
	var once sync.Once
	var pullErr error
	for range concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := b.runSlot(ctx, p); err != nil {
				once.Do(func() { pullErr = err })
				stopPulling()
			}
		}()
	}
	wg.Wait()

	if pullErr != nil && b.GetRetry() {
		b.GetRetryFunc()(b.GetRetryStopChan())
	}
	if b.GetRetry() {
		return true, fmt.Errorf("pull from %s: %w", b.subscription, pullErr)
	}
	close(b.stopped)
	return false, nil
}

// runSlot pulls and runs one job at a time until ctx ends or a pull fails.
func (b *pubsubBroker) runSlot(ctx context.Context, p iface.TaskProcessor) error {
	for {
		res, err := b.client.SubscriptionAdminClient.Pull(ctx, &pubsubpb.PullRequest{
			Subscription: b.subscription,
			MaxMessages:  1,
		})
		switch {
		case ctx.Err() != nil:
			return nil
		case status.Code(err) == codes.DeadlineExceeded:
			continue // nothing arrived within the long poll
		case err != nil:
			return err
		}
		for _, m := range res.ReceivedMessages {
			b.handle(m, p)
		}
	}
}

// handle runs one job while keeping its lease, then acks or nacks it. The slot
// stays busy until the broker took that answer or the lease has lapsed, so a
// slot never holds two leased messages.
func (b *pubsubBroker) handle(m *pubsubpb.ReceivedMessage, p iface.TaskProcessor) {
	fields := log.Fields{"message_id": m.Message.MessageId}
	started := time.Now()
	// confirmed is the deadline set by the last extension that answered. Until
	// one answers, the subscription's own deadline holds, which may be long, so
	// a crash would hide the job for that long. unconfirmed is the latest
	// deadline a failed extension may still have set: its response may be lost
	// after the broker applied it, at the latest when its call timed out.
	confirmed, unconfirmed := started.Add(maxAckDeadline), time.Time{}
	leasedUntil := func() time.Time {
		if unconfirmed.After(confirmed) {
			return unconfirmed
		}
		return confirmed
	}
	extend := func() {
		sent := time.Now()
		if b.setDeadline(m.AckId, lease) == nil {
			confirmed = time.Now().Add(lease)
		} else if until := sent.Add(rpcTimeout + lease); until.After(unconfirmed) {
			unconfirmed = until
		}
	}
	extend()

	answered := make(chan bool, 1)
	go func() { answered <- b.consumeOne(m.Message, p) }()
	ticker := time.NewTicker(lease / 3)
	defer ticker.Stop()
	for {
		select {
		case ok := <-answered:
			b.settle(fields, m.AckId, leasedUntil(), ok)
			return
		case <-b.release:
			log.WithFields(fields).Warn("Grader stopping, running job released for redelivery")
			b.settle(fields, m.AckId, leasedUntil(), false)
			return
		case <-ticker.C:
			if time.Since(started) < maxJobDuration {
				extend()
			}
		}
	}
}

// settle acks (or nacks) the message until that lands, the lease has lapsed or
// the grader stops. Once the lease lapsed the message is no longer this
// slot's, so it may pull again. A stopping grader nacks once, best effort, and
// otherwise lets the lease lapse, so shutdown never waits on a failing broker.
func (b *pubsubBroker) settle(fields log.Fields, ackID string, leasedUntil time.Time, ack bool) {
	what, answer := "nack", func() error { return b.setDeadline(ackID, 0) }
	if ack {
		what, answer = "ack", func() error { return b.acknowledge(ackID) }
	}
	for {
		if !time.Now().Before(leasedUntil) {
			log.WithFields(fields).Errorf("The %s did not land before the lease lapsed, the job may be graded again", what)
			return
		}
		err := answer()
		if err == nil {
			return
		}
		log.WithFields(fields).WithError(err).Warnf("The %s failed, retrying", what)
		select {
		case <-time.After(settleRetry):
		case <-b.release:
			if ack {
				_ = b.setDeadline(ackID, 0)
			}
			log.WithFields(fields).Warnf("Grader stopping, the %s is abandoned and the job may be graded again", what)
			return
		}
	}
}

func (b *pubsubBroker) acknowledge(ackID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()
	return b.client.SubscriptionAdminClient.Acknowledge(ctx, &pubsubpb.AcknowledgeRequest{
		Subscription: b.subscription, AckIds: []string{ackID},
	})
}

// setDeadline extends a lease, or with 0 nacks the message for redelivery now.
func (b *pubsubBroker) setDeadline(ackID string, d time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()
	err := b.client.SubscriptionAdminClient.ModifyAckDeadline(ctx, &pubsubpb.ModifyAckDeadlineRequest{
		Subscription: b.subscription, AckIds: []string{ackID}, AckDeadlineSeconds: int32(d / time.Second),
	})
	if err != nil {
		log.WithError(err).Warnf("Set ack deadline to %s failed", d)
	}
	return err
}

// StopConsuming stops pulling and waits shutdownGrace for running jobs. Jobs
// still running then are nacked, so another grader takes them at once.
func (b *pubsubBroker) StopConsuming() {
	b.Broker.StopConsuming()
	select {
	case <-b.stopped:
	case <-time.After(shutdownGrace):
		close(b.release)
		<-b.stopped
	}
	if err := b.client.Close(); err != nil {
		log.WithError(err).Warn("Close Pub/Sub client")
	}
}

// consumeOne runs one job and reports whether its message may be acked.
func (b *pubsubBroker) consumeOne(msg *pubsubpb.PubsubMessage, p iface.TaskProcessor) bool {
	fields := log.Fields{"message_id": msg.MessageId}
	signature := new(tasks.Signature)
	decoder := json.NewDecoder(bytes.NewReader(msg.Data))
	decoder.UseNumber()
	if err := decoder.Decode(signature); err != nil {
		log.WithFields(fields).WithError(err).Error("Undecodable message dropped")
		return true
	}
	if !b.IsTaskRegistered(signature.Name) {
		log.WithFields(fields).WithField("task", signature.Name).Error("Unknown task dropped")
		return true
	}
	if err := p.Process(signature); err != nil {
		log.WithField("uuid", signature.UUID).WithError(err).Warn("Task processing error")
	}
	_, unanswered := b.unanswered.LoadAndDelete(signature)
	return !unanswered
}

// Publish sends a signature to PUBSUB_TOPIC_NAME. Only cmd/send publishes.
func (b *pubsubBroker) Publish(ctx context.Context, signature *tasks.Signature) error {
	if b.topic == "" {
		return errors.New("PUBSUB_TOPIC_NAME is required to publish")
	}
	data, err := json.Marshal(signature)
	if err != nil {
		return fmt.Errorf("encode signature: %w", err)
	}
	publisher := b.client.Publisher(b.topic)
	defer publisher.Stop()
	_, err = publisher.Publish(ctx, &pubsub.Message{Data: data}).Get(ctx)
	return err
}
