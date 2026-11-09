package machinery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"

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
	release chan struct{}
	stopped chan struct{}
}

func newAMQPBroker(cnf *config.Config, release chan struct{}) *amqpBroker {
	return &amqpBroker{Broker: amqpbroker.New(cnf).(*amqpbroker.Broker), release: release, stopped: make(chan struct{})}
}

// StartConsuming runs up to concurrency jobs at once until StopConsuming, or
// until the connection fails. Machinery then calls it again.
func (b *amqpBroker) StartConsuming(consumerTag string, concurrency int, p iface.TaskProcessor) (bool, error) {
	b.Broker.Broker.StartConsuming(consumerTag, concurrency, p)
	cnf := b.GetConfig()
	conn, channel, queue, _, closed, err := b.Connect(cnf.Broker, cnf.MultipleBrokerSeparator, cnf.TLSConfig,
		cnf.AMQP.Exchange, cnf.AMQP.ExchangeType, cnf.DefaultQueue, true, false, cnf.AMQP.BindingKey,
		nil, amqp.Table(cnf.AMQP.QueueDeclareArgs), amqp.Table(cnf.AMQP.QueueBindingArgs))
	if err != nil {
		b.GetRetryFunc()(b.GetRetryStopChan())
		return b.end(err)
	}
	// Closing the channel requeues every delivery still unacked.
	defer func() { _ = b.Close(channel, conn) }()
	if err := channel.Qos(concurrency, 0, false); err != nil {
		return b.end(fmt.Errorf("channel qos: %w", err))
	}
	deliveries, err := channel.Consume(queue.Name, consumerTag, false, false, false, false, nil)
	if err != nil {
		return b.end(fmt.Errorf("consume %s: %w", queue.Name, err))
	}

	var jobs sync.WaitGroup
	for {
		select {
		case amqpErr := <-closed:
			return b.end(fmt.Errorf("connection closed: %v", amqpErr))
		case d, ok := <-deliveries:
			if !ok {
				return b.end(fmt.Errorf("delivery channel of %s closed", queue.Name))
			}
			jobs.Add(1)
			go func() {
				defer jobs.Done()
				b.consumeOne(d, p)
			}()
		case <-b.GetStopChan():
			_ = channel.Cancel(consumerTag, false) // stop pulling, the running jobs go on
			jobs.Wait()
			return b.end(nil)
		}
	}
}

// end returns from StartConsuming. Once StopConsuming has turned retry off,
// Machinery calls StartConsuming no more, so stopped closes exactly once.
func (b *amqpBroker) end(err error) (bool, error) {
	if b.GetRetry() {
		return true, err
	}
	close(b.stopped)
	return false, nil
}

// StopConsuming stops pulling and waits shutdownGrace for running jobs. Jobs
// still running then are nacked, so another grader takes them at once.
func (b *amqpBroker) StopConsuming() {
	b.Broker.Broker.StopConsuming()
	stopWithGrace(b.stopped, b.release)
}

// consumeOne runs one job, then acks its message, or nacks it back to the
// queue if the grader released the job meanwhile. Undecodable and unknown
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
	select {
	case <-b.release:
		err = d.Nack(false, true)
	default:
		err = d.Ack(false)
	}
	if err != nil {
		// The broker requeues an unanswered delivery when the channel closes.
		log.WithField("uuid", signature.UUID).WithError(err).Warn("Ack or nack failed")
	}
}
