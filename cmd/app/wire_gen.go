// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//+build !wireinject

package main

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/lestrrat-go/file-rotatelogs"
	"go-scaffold/internal/app"
	"go-scaffold/internal/app/command"
	greet3 "go-scaffold/internal/app/command/handler/greet"
	"go-scaffold/internal/app/command/script"
	"go-scaffold/internal/app/component/data"
	"go-scaffold/internal/app/component/discovery/consul"
	"go-scaffold/internal/app/component/discovery/etcd"
	"go-scaffold/internal/app/component/orm"
	"go-scaffold/internal/app/component/redis"
	"go-scaffold/internal/app/component/trace"
	config2 "go-scaffold/internal/app/config"
	"go-scaffold/internal/app/cron"
	"go-scaffold/internal/app/cron/job"
	"go-scaffold/internal/app/repository/user"
	"go-scaffold/internal/app/service/v1/greet"
	user2 "go-scaffold/internal/app/service/v1/user"
	"go-scaffold/internal/app/transport"
	"go-scaffold/internal/app/transport/grpc"
	"go-scaffold/internal/app/transport/http"
	greet2 "go-scaffold/internal/app/transport/http/handler/v1/greet"
	trace2 "go-scaffold/internal/app/transport/http/handler/v1/trace"
	user3 "go-scaffold/internal/app/transport/http/handler/v1/user"
	"go-scaffold/internal/app/transport/http/router"
	"go.uber.org/zap"
)

// Injectors from wire.go:

func initApp(rotateLogs *rotatelogs.RotateLogs, logLogger log.Logger, zapLogger *zap.Logger, configConfig *config2.Config, config3 *orm.Config, config4 *data.Config, config5 *redis.Config, config6 *trace.Config, etcdConfig *etcd.Config, consulConfig *consul.Config) (*app.App, func(), error) {
	db, cleanup2, err := orm.New(config3, logLogger, zapLogger)
	if err != nil {
		return nil, nil, err
	}
	client, cleanup3, err := redis.New(config5, logLogger)
	if err != nil {
		cleanup2()
		return nil, nil, err
	}
	example := job.NewExample(logLogger)
	cronCron, err := cron.New(logLogger, db, client, example)
	if err != nil {
		cleanup3()
		cleanup2()
		return nil, nil, err
	}
	service := greet.NewService(logLogger, configConfig)
	handler := greet2.New(logLogger, zapLogger, configConfig, service)
	tracer, cleanup4, err := trace.New(config6, logLogger)
	if err != nil {
		cleanup3()
		cleanup2()
		return nil, nil, err
	}
	traceHandler := trace2.New(logLogger, configConfig, tracer, service)
	repository := user.New(db, client)
	userService := user2.NewService(logLogger, configConfig, repository)
	userHandler := user3.New(logLogger, userService)
	engine := router.New(rotateLogs, zapLogger, configConfig, handler, traceHandler, userHandler)
	server := http.NewServer(logLogger, configConfig, engine)
	grpcServer := grpc.NewServer(logLogger, configConfig, service, userService)
	registry, err := etcd.New(etcdConfig, zapLogger)
	if err != nil {
		cleanup4()
		cleanup3()
		cleanup2()
		return nil, nil, err
	}
	transportTransport := transport.New(logLogger, configConfig, server, grpcServer, registry)
	appApp := app.New(logLogger, configConfig, db, cronCron, transportTransport)
	return appApp, func() {
		cleanup4()
		cleanup3()
		cleanup2()
	}, nil
}

func initCommand(rotateLogs *rotatelogs.RotateLogs, logLogger log.Logger, zapLogger *zap.Logger, configConfig *config2.Config, config3 *orm.Config, config4 *data.Config, config5 *redis.Config, config6 *trace.Config, etcdConfig *etcd.Config, consulConfig *consul.Config) (*command.Command, func(), error) {
	handler := greet3.NewHandler(logLogger)
	s0000000000 := script.NewS0000000000(logLogger)
	commandCommand := command.New(handler, s0000000000)
	return commandCommand, func() {
	}, nil
}
