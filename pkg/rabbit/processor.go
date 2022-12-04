package rabbit

import "github.com/streadway/amqp"

const (
	ExchangeKindTopic   = "topic"
	ExchangeKindFanout  = "fanout"
	ExchangeKindDirect  = "direct"
	ExchangeKindHeaders = "headers"
)

type QueueDeclarationValues struct {
	QueueName  string
	Durable    bool
	AutoDelete bool

	Exclusive bool
	NoWait    bool
	Args      amqp.Table
}

type ExchangeDeclarationValues struct {
	ExchangeName string
	ExchangeKind string
	Durable      bool
	AutoDelete   bool
	Internal     bool
	NoWait       bool
	Args         amqp.Table
}

type QueueBindValues struct {
	QueueName    string
	RoutingKey   string
	ExchangeName string
	NoWait       bool
	Args         amqp.Table
}

type processor struct {
	ch *amqp.Channel
}

/*
QueueDeclare declares a queue to hold messages and deliver to consumers.
Declaring creates a queue if it doesn't already exist, or ensures that an
existing queue matches the same parameters.

Every queue declared gets a default binding to the empty exchange "" which has
the type "direct" with the routing key matching the queue's name.  With this
default binding, it is possible to publish messages that route directly to
this queue by publishing to "" with the routing key of the queue name.

	QueueDeclare(...)
	Publish(...)

	Delivery       Exchange  Key       Queue
	-----------------------------------------------
	key: alerts -> ""     -> alerts -> alerts

The queue name may be empty, in which case the server will generate a unique name
which will be returned to the Name field of Queue struct.

Durable and Non-Auto-Deleted queues will survive server restarts and remain
when there are no remaining consumers or bindings.  Persistent Publishings will
be restored in this queue on server restart.  These queues are only able to be
bound to durable exchanges.

Non-Durable and Auto-Deleted queues will not be redeclared on server restart
and will be deleted by the server after a short time when the last consumer is
canceled or the last consumer's channel is closed.  Queues with this lifetime
can also be deleted normally with QueueDelete.  These durable queues can only
be bound to non-durable exchanges.

Non-Durable and Non-Auto-Deleted queues will remain declared as long as the
server is running regardless of how many consumers.  This lifetime is useful
for temporary topologies that may have long delays between consumer activity.
These queues can only be bound to non-durable exchanges.

Durable and Auto-Deleted queues will be restored on server restart, but without
active consumers will not survive and be removed.  This Lifetime is unlikely
to be useful.

Exclusive queues are only accessible by the connection that declares them and
will be deleted when the connection closes.  Channels on other connections
will receive an error when attempting  to declare, bind, consume, purge or
delete a queue with the same name.

When noWait is true, the queue will assume to be declared on the server.  A
channel exception will arrive if the conditions are met for existing queues
or attempting to modify an existing queue from a different connection.

When the error return value is not nil, you can assume the queue could not be
declared with these parameters, and the channel will be closed.
*/
func (p *processor) QueueDeclare(values QueueDeclarationValues) error {
	_, err := p.ch.QueueDeclare(
		values.QueueName,
		values.Durable,
		values.AutoDelete,
		values.Exclusive,
		values.NoWait,
		values.Args,
	)

	return err
}

/*
ExchangeDeclare declares an exchange on the server. If the exchange does not
already exist, the server will create it.  If the exchange exists, the server
verifies that it is of the provided type, durability and auto-delete flags.

Errors returned from this method will close the channel.

Exchange names starting with "amq." are reserved for pre-declared and
standardized exchanges. The client MAY declare an exchange starting with
"amq." if the passive option is set, or the exchange already exists.  Names can
consist of a non-empty sequence of letters, digits, hyphen, underscore,
period, or colon.

Each exchange belongs to one of a set of exchange kinds/types implemented by
the server. The exchange types define the functionality of the exchange - i.e.
how messages are routed through it. Once an exchange is declared, its type
cannot be changed.  The common types are "direct", "fanout", "topic" and
"headers".

Durable and Non-Auto-Deleted exchanges will survive server restarts and remain
declared when there are no remaining bindings.  This is the best lifetime for
long-lived exchange configurations like stable routes and default exchanges.

Non-Durable and Auto-Deleted exchanges will be deleted when there are no
remaining bindings and not restored on server restart.  This lifetime is
useful for temporary topologies that should not pollute the virtual host on
failure or after the consumers have completed.

Non-Durable and Non-Auto-deleted exchanges will remain as long as the server is
running including when there are no remaining bindings.  This is useful for
temporary topologies that may have long delays between bindings.

Durable and Auto-Deleted exchanges will survive server restarts and will be
removed before and after server restarts when there are no remaining bindings.
These exchanges are useful for robust temporary topologies or when you require
binding durable queues to auto-deleted exchanges.

Note: RabbitMQ declares the default exchange types like 'amq.fanout' as
durable, so queues that bind to these pre-declared exchanges must also be
durable.

Exchanges declared as `internal` do not accept publishings. Internal
exchanges are useful when you wish to implement inter-exchange topologies
that should not be exposed to users of the broker.

When noWait is true, declare without waiting for a confirmation from the server.
The channel may be closed as a result of an error.  Add a NotifyClose listener
to respond to any exceptions.

Optional amqp.Table of arguments that are specific to the server's implementation of
the exchange can be sent for exchange types that require extra parameters.
*/
func (p *processor) ExchangeDeclare(values ExchangeDeclarationValues) error {
	return p.ch.ExchangeDeclare(
		values.ExchangeName,
		values.ExchangeKind,
		values.Durable,
		values.AutoDelete,
		values.Internal,
		values.NoWait,
		values.Args,
	)
}

/*
QueueBind binds an exchange to a queue so that publishings to the exchange will
be routed to the queue when the publishing routing key matches the binding
routing key.

	QueueBind("pagers", "alert", "log", false, nil)
	QueueBind("emails", "info", "log", false, nil)

	Delivery       Exchange  Key       Queue
	-----------------------------------------------
	key: alert --> log ----> alert --> pagers
	key: info ---> log ----> info ---> emails
	key: debug --> log       (none)    (dropped)

If a binding with the same key and arguments already exists between the
exchange and queue, the attempt to rebind will be ignored and the existing
binding will be retained.

In the case that multiple bindings may cause the message to be routed to the
same queue, the server will only route the publishing once.  This is possible
with topic exchanges.

	QueueBind("pagers", "alert", "amq.topic", false, nil)
	QueueBind("emails", "info", "amq.topic", false, nil)
	QueueBind("emails", "#", "amq.topic", false, nil) // match everything

	Delivery       Exchange        Key       Queue
	-----------------------------------------------
	key: alert --> amq.topic ----> alert --> pagers
	key: info ---> amq.topic ----> # ------> emails
	                         \---> info ---/
	key: debug --> amq.topic ----> # ------> emails

It is only possible to bind a durable queue to a durable exchange regardless of
whether the queue or exchange is auto-deleted.  Bindings between durable queues
and exchanges will also be restored on server restart.

If the binding could not complete, an error will be returned and the channel
will be closed.

When noWait is false and the queue could not be bound, the channel will be
closed with an error.
*/
func (p *processor) QueueBind(values QueueBindValues) error {
	return p.ch.QueueBind(
		values.QueueName,
		values.RoutingKey,
		values.ExchangeName,
		values.NoWait,
		values.Args,
	)
}
