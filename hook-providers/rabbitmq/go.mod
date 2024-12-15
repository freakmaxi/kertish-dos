module github.com/freakmaxi/kertish-dos/hooks-providers/rabbitmq

go 1.21

require (
	github.com/freakmaxi/kertish-dos/basics v0.0.0-20241109084023-61da6111a48a
	github.com/streadway/amqp v1.1.0
)

require (
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
)

replace github.com/freakmaxi/kertish-dos/basics => ../../basics
