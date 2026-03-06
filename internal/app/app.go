package app

import (
	"IOTProject/internal/config"
	"IOTProject/internal/repository/postgres"
	"IOTProject/internal/router"
	"IOTProject/internal/service"
	"IOTProject/internal/transport/http"
	"IOTProject/pkg/db"
	myredis "IOTProject/pkg/redis"
	"IOTProject/pkg/utils"
	"fmt"

	"github.com/redis/go-redis/v9"
	gormpg "gorm.io/driver/postgres" // 给 GORM 驱动起别名
	"gorm.io/gorm"
)

type App struct {
	cfg         *config.Config
	dbgorm      *gorm.DB
	redisClient *redis.Client
	locker      utils.Locker
}

func New() (*App, func(), error) {
	cfg := config.Load()
	pool, err := db.NewPostgres(cfg.GetDSN())
	if err != nil {
		return nil, nil, fmt.Errorf("初始化数据库失败:%w", err)
	}
	rdb, err := myredis.InitRedis("localhost:6379", "")
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("初始化 Redis 失败:%w", err)
	}
	gormDb, err := gorm.Open(gormpg.Open(cfg.GetDSN()), &gorm.Config{})
	if err != nil {
		return nil, nil, fmt.Errorf("GORM初始化失败: %w", err)
	}

	locker := utils.NewRedisLocker(rdb)
	cleanup := func() {
		pool.Close()
		if err := rdb.Close(); err != nil {
			fmt.Printf("Redis 关闭时出错: %v\n", err)
		}

		fmt.Println("数据库与 Redis 连接已关闭")
	}
	fmt.Println("所有依赖项注入完成")
	return &App{
		cfg:         cfg,
		dbgorm:      gormDb,
		redisClient: rdb,
		locker:      locker,
	}, cleanup, nil
}
func (a *App) Run() error {
	utils.InitJWT(a.cfg.JWTSecret)
	userRepo := postgres.NewUserRepository(a.dbgorm)
	userSvc := service.NewUserService(userRepo, a.redisClient, a.locker)
	userHdl := http.NewUserHandler(*userSvc)

	deviceRepo := postgres.NewDeviceRepository(a.dbgorm)
	deviceSvc := service.NewDeviceService(deviceRepo)
	deviceHdl := http.NewDeviceHandle(*deviceSvc)

	devicedataRepo := postgres.NewDeviceDataRepository(a.dbgorm)
	devicedataSvc := service.NewDeviceDataService(devicedataRepo, deviceRepo)
	devicedataHdl := http.NewDeviceDataHandle(*devicedataSvc)

	r := router.InitRouter(userHdl, deviceHdl, devicedataHdl, userRepo)
	fmt.Printf("系统启动成功，监听端口: %s\n", a.cfg.ServerPort)
	return r.Run(":" + a.cfg.ServerPort)

}
