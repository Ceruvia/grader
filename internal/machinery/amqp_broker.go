package machinery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqpbroker "github.com/RichardKnop/machinery/v2/brokers/amqp"
	"github.com/RichardKnop/machinery/v2/brokers/iface"
	"github.com/RichardKnop/machinery/v2/config"
	"github.com/RichardKnop/machinery/v2/tasks"
	amqp "github.com/rabbitmq/amqp091-go"
	log "github.com/sirupsen/logrus"
)

// amqpBroker consumes like Machinery's AMQP broker, which it embeds for the
// connection and Publish. Machinery acks every job it ran and waits for running
// jobs without a bound. This broker gives running jobs shutdownGrace, then
// nacks the ones it released back to the queue, so another grader takes them.
type amqpBroker struct {
	*amqpbroker.Broker
	*stopState
}

func newAMQPBroker(cnf *config.Config, stop *stopState) *amqpBroker {
	return &amqpBroker{Broker: amqpbroker.New(cnf).(*amqpbroker.Broker), stopState: stop}
}

// StartConsuming runs up to concurrency jobs at once until StopConsuming, or
// until the connection fails. Machinery then calls it again. Once
// StopConsuming has begun, Machinery calls it no more, so stopped closes once.
func (b *amqpBroker) StartConsuming(consumerTag string, concurrency int, p iface.TaskProcessor) (bool, error) {
	b.Broker.Broker.StartConsuming(consumerTag, concurrency, p)
	err := b.consume(consumerTag, concurrency, p)
	if !b.stopping.Load() {
		return true, err
	}
	close(b.stopped)
	return false, nil
}

// consume returns once the connection fails, or once the stop is done and the
// channel closed, which requeues every delivery still unacked.
func (b *amqpBroker) consume(consumerTag string, concurrency int, p iface.TaskProcessor) error {
	cnf := b.GetConfig()
	conn, channel, queue, _, closed, err := b.Connect(cnf.Broker, cnf.MultipleBrokerSeparator, cnf.TLSConfig,
		cnf.AMQP.Exchange, cnf.AMQP.ExchangeType, cnf.DefaultQueue, true, false, cnf.AMQP.BindingKey,
		nil, amqp.Table(cnf.AMQP.QueueDeclareArgs), amqp.Table(cnf.AMQP.QueueBindingArgs))
	if err != nil {
		b.GetRetryFunc()(b.GetRetryStopChan())
		return err
	}
	defer func() { _ = b.Close(channel, conn) }()
	if err := channel.Qos(concurrency, 0, false); err != nil {
		return fmt.Errorf("channel qos: %w", err)
	}
	deliveries, err := channel.Consume(queue.Name, consumerTag, false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume %s: %w", queue.Name, err)
	}

	var jobs sync.WaitGroup
	for {
		select {
		case amqpErr := <-closed:
			return fmt.Errorf("connection closed: %v", amqpErr)
		case d, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("delivery channel of %s closed", queue.Name)
			}
			jobs.Add(1)
			go func() {
				defer jobs.Done()
				b.consumeOne(d, p)
			}()
		case <-b.GetStopChan():
			_ = channel.Cancel(consumerTag, false) // stop pulling, the running jobs go on
			ended := make(chan struct{})
			go func() { jobs.Wait(); close(ended) }()
			select {
			case <-ended:
			case <-b.released.Done():
				// The released jobs' runs are being killed: wait for them, but not forever.
				select {
				case <-ended:
				case <-time.After(shutdownGrace):
					log.Error("Released jobs did not end within the grace, exiting anyway")
				}
			}
			return nil
		}
	}
}

// StopConsuming stops pulling and waits shutdownGrace for running jobs. Jobs
// still running then are nacked, so another grader takes them at once.
func (b *amqpBroker) StopConsuming() {
	b.stopping.Store(true)
	b.Broker.Broker.StopConsuming()
	b.waitStopped()
}

// consumeOne runs one job, then acks its message once the Backend gave its
// final answer, and otherwise nacks it back to the queue. Undecodable and unknown
// tasks are handled as Machinery's AMQP broker does.
func (b *amqpBroker) consumeOne(d amqp.Delivery, p iface.TaskProcessor) {
	signature := new(tasks.Signature)
	decoder := json.NewDecoder(bytes.NewReader(d.Body))
	decoder.UseNumber()
	if err := decoder.Decode(signature); err != nil {
		log.WithError(err).Error("Undecodable message dropped")
		_ = d.Nack(false, false)
		return
	}
	if !b.IsTaskRegistered(signature.Name) {
		log.WithField("task", signature.Name).Warn("Task not registered with this worker, requeued")
		_ = d.Nack(false, !signature.IgnoreWhenTaskNotRegistered)
		return
	}
	if err := p.Process(signature); err != nil {
		log.WithField("uuid", signature.UUID).WithError(err).Warn("Task processing error")
	}
	var err error
	if b.answered(signature) {
		err = d.Ack(false)
	} else {
		err = d.Nack(false, true)
	}
	if err != nil {
		// The broker requeues an unanswered delivery when the channel closes.
		log.WithField("uuid", signature.UUID).WithError(err).Warn("Ack or nack failed")
	}
}
