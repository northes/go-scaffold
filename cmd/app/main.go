package main

import (
	"context"
	"github.com/go-kratos/kratos/contrib/config/apollo/v2"
	kzap "github.com/go-kratos/kratos/contrib/log/zap/v2"
	kconfig "github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/thoas/go-funk"
	"go-scaffold/internal/app/command"
	appconfig "go-scaffold/internal/app/config"
	"go-scaffold/pkg/log"
	"go-scaffold/pkg/path"
	"go.uber.org/zap"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

var (
	appName     = "go-scaffold"
	hostname, _ = os.Hostname()
)

var (
	rootPath              = path.RootPath()
	logPath               string // 日志输出路径
	logLevel              string // 日志等级
	logFormat             string // 日志输出格式
	logCallerSkip         int    // 日志 caller 跳过层数
	configPath            string // 配置文件路径
	apolloConfigEnable    bool   // apollo 是否启用
	apolloConfigEndpoint  string // apollo 连接地址
	apolloConfigAppID     string // apollo appID
	apolloConfigCluster   string // apollo cluster
	apolloConfigNamespace string // apollo 命名空间
	apolloConfigSecret    string // apollo 密钥
)

func init() {
	pflag.StringVarP(&logPath, "log.path", "", "logs/%Y%m%d.log", "日志输出路径")
	pflag.StringVarP(&logLevel, "log.level", "", "info", "日志等级（debug、info、warn、error、panic、fatal）")
	pflag.StringVarP(&logFormat, "log.format", "", "json", "日志输出格式（text、json）")
	pflag.IntVarP(&logCallerSkip, "log.caller-skip", "", 4, "日志 caller 跳过层数")
	pflag.StringVarP(&configPath, "config", "f", filepath.Join(rootPath, "etc/config.yaml"), "配置文件路径")
	pflag.BoolVarP(&apolloConfigEnable, "config.apollo.enable", "", false, "apollo 是否启用")
	pflag.StringVarP(&apolloConfigEndpoint, "config.apollo.endpoint", "", "http://localhost:8080", "apollo 连接地址")
	pflag.StringVarP(&apolloConfigAppID, "config.apollo.appid", "", "", "apollo appID")
	pflag.StringVarP(&apolloConfigCluster, "config.apollo.cluster", "", "default", "apollo cluster")
	pflag.StringVarP(&apolloConfigNamespace, "config.apollo.namespace", "", "application", "apollo 命名空间")
	pflag.StringVarP(&apolloConfigSecret, "config.apollo.secret", "", "", "apollo 密钥")

	cobra.OnInitialize(setup)
}

var (
	loggerWriter *rotatelogs.RotateLogs // 日志全局 Writer
	logger       klog.Logger            // 日志实例
	hLogger      *klog.Helper           // 日志实例
	zLogger      *zap.Logger            // zap 日志实例
	config       kconfig.Config
	configModel  = new(appconfig.Config) // app 配置实例
)

func main() {
	defer cleanup()

	cmd := &cobra.Command{
		Use: "app",
		Run: func(cmd *cobra.Command, args []string) {
			hLogger.Info("starting app ...")

			// 监听退出信号
			signalCtx, signalStop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer signalStop()

			appServ, appCleanup, err := initApp(loggerWriter, logger, zLogger, configModel)
			if err != nil {
				panic(err)
			}
			defer appCleanup()
			// 调用 app 启动钩子
			if err := appServ.Start(signalStop); err != nil {
				panic(err)
			}

			// 等待退出信号
			<-signalCtx.Done()
			signalStop()

			hLogger.Info("the app is shutting down ...")

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(configModel.App.Timeout)*time.Second)
			defer cancel()

			// 关闭应用
			if err := appServ.Stop(ctx); err != nil {
				panic(err)
			}
		},
	}

	command.Setup(cmd, func() (*command.Command, func(), error) {
		return initCommand(loggerWriter, logger, zLogger, configModel)
	})

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

func setup() {
	var err error

	if logPath != "" {
		if !filepath.IsAbs(logPath) {
			logPath = filepath.Join(path.RootPath(), logPath)
		}

		loggerWriter, err = rotatelogs.New(
			logPath,
			rotatelogs.WithClock(rotatelogs.Local),
		)
		if err != nil {
			panic(err)
		}
	}

	// 日志初始化
	var writer io.Writer
	if loggerWriter == nil {
		writer = os.Stdout
	} else {
		writer = io.MultiWriter(os.Stdout, loggerWriter)
	}
	zLogger = log.New(
		log.WithLevel(log.Level(logLevel)),
		log.WithFormat(log.Format(logFormat)),
		log.WithWriter(writer),
		log.WithCallerSkip(logCallerSkip),
	)
	logger = klog.With(
		kzap.NewLogger(zLogger),
		"service.id", hostname,
		"service.name", appName,
		"trace_id", tracing.TraceID(),
		"span_id", tracing.SpanID(),
	)
	hLogger = klog.NewHelper(logger)

	hLogger.Info("initializing resource ...")
	hLogger.Infof("the log output directory: %s", filepath.Dir(logPath))

	// 加载配置
	if configPath == "" {
		panic("config path is missing")
	}

	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(path.RootPath(), configPath)
	}

	hLogger.Infof("load config from: %s", configPath)

	configResources := []kconfig.Source{file.NewSource(configPath)}
	if apolloConfigEnable { // 启用 apollo
		hLogger.Infof("enable remote config, config will be loaded from remote config center")

		configResources = append(configResources, apollo.NewSource(
			apollo.WithEndpoint(apolloConfigEndpoint),
			apollo.WithAppID(apolloConfigAppID),
			apollo.WithCluster(apolloConfigCluster),
			apollo.WithNamespace(apolloConfigNamespace),
			apollo.WithSecret(apolloConfigSecret),
		))
	}

	config = kconfig.New(
		kconfig.WithSource(configResources...),
		kconfig.WithLogger(logger),
	)

	if err := config.Load(); err != nil {
		panic(err)
	}

	if err := config.Scan(configModel); err != nil {
		panic(err)
	}

	if err := appconfig.Loaded(logger, config, configModel); err != nil {
		panic(err)
	}

	// 检查环境是否设置正确
	if !funk.ContainsString(appconfig.SupportedEnvs, configModel.App.Env.String()) {
		panic("unsupported env value: " + configModel.App.Env)
	}

	hLogger.Infof("current env: %s", configModel.App.Env)
}

// cleanup 资源回收
func cleanup() {
	if hLogger != nil {
		hLogger.Info("resource cleaning ...")
	}

	if config != nil {
		if err := config.Close(); err != nil {
			hLogger.Error(err.Error())
		}
	}

	if loggerWriter != nil {
		if err := loggerWriter.Close(); err != nil {
			panic(err)
		}
	}

	if zLogger != nil {
		if err := zLogger.Sync(); err != nil {
			hLogger.Error(err.Error())
		}
	}
}
