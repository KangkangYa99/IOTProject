package app

import (
	"IOTProject/internal/config"
	"IOTProject/internal/domain"
	"IOTProject/internal/repository/postgres"
	"IOTProject/internal/router"
	"IOTProject/internal/service"
	"IOTProject/internal/transport/http"
	"IOTProject/internal/websocket"
	myredis "IOTProject/pkg/redis"
	"IOTProject/pkg/utils"
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type App struct {
	cfg         *config.Config
	gorm        *gorm.DB
	redisClient *redis.Client
	locker      utils.Locker
	Hub         *websocket.Hub
}

// New 初始化应用基础依赖
func New() (*App, func(), error) {
	cfg := config.Load()

	//初始化 GORM
	db, err := initGORM(cfg)
	if err != nil {
		return nil, nil, err
	}

	//初始化 Redis
	rdb, err := myredis.InitRedis(cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("Redis初始化失败: %w", err)
	}

	//初始化业务组件
	locker := utils.NewRedisLocker(rdb)
	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
		_ = rdb.Close()
		fmt.Println("已清理所有底层连接")
	}

	return &App{
		cfg:         cfg,
		gorm:        db,
		redisClient: rdb,
		locker:      locker,
	}, cleanup, nil
}

// initGORM 专门处理数据库连接逻辑
func initGORM(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(gormpg.Open(cfg.GetDSN()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("GORM开启失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("SQL实例获取失败: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 健康检查
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("数据库Ping失败: %w", err)
	}

	return db, nil
}

func (a *App) Run() error {
	utils.InitJWT(a.cfg.JWTSecret)
	repos := a.initRepositories()

	a.Hub = websocket.NewHub(nil)
	go a.Hub.Run()

	// Service层
	svcs := a.initServices(repos, a.Hub)

	eventHandler := service.NewDeviceEventHandler(svcs.data, repos.device)

	a.Hub.SetEventListener(eventHandler)

	// Handler层与路由
	wsHandler := websocket.NewWsHandler(a.Hub, eventHandler)
	r := router.InitRouter(router.HandlerConfig{
		UserHandle:         http.NewUserHandler(*svcs.user),
		DeviceHandle:       http.NewDeviceHandle(*svcs.device, a.Hub),
		DeviceDataHandle:   http.NewDeviceDataHandle(*svcs.data),
		DevicePolicyHandle: http.NewDevicePolicyHandler(*svcs.policy, repos.device, a.Hub),
		UserRepo:           repos.user,
		WsHandler:          wsHandler,
	})
	fmt.Printf("系统初始化完成，启动端口: %s\n", a.cfg.ServerPort)
	return r.Run(":" + a.cfg.ServerPort)
}

type repositoryBundle struct {
	user   domain.UserRepository
	device domain.DeviceRepository
	policy domain.DevicePolicyRepository
	data   domain.DeviceDataRepository
}

func (a *App) initRepositories() repositoryBundle {
	return repositoryBundle{
		user:   postgres.NewUserRepository(a.gorm),
		device: postgres.NewDeviceRepository(a.gorm, a.redisClient),
		policy: postgres.NewDevicePolicyRepository(a.gorm),
		data:   postgres.NewDeviceDataRepository(a.gorm),
	}
}

type serviceBundle struct {
	user   *service.UserService
	device *service.DeviceService
	policy *service.DevicePolicyService
	data   *service.DeviceDataService
}

func (a *App) initServices(repos repositoryBundle, hub *websocket.Hub) serviceBundle {
	return serviceBundle{
		user:   service.NewUserService(repos.user, a.redisClient, a.locker),
		device: service.NewDeviceService(repos.device),
		policy: service.NewDevicePolicyService(repos.policy, repos.device),
		data:   service.NewDeviceDataService(repos.data, repos.device, repos.policy, hub, a.redisClient),
	}
}
