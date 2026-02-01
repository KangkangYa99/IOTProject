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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type App struct {
	cfg         *config.Config
	dbPool      *pgxpool.Pool
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
		dbPool:      pool,
		redisClient: rdb,
		locker:      locker,
	}, cleanup, nil
}
func (a *App) Run() error {
	utils.InitJWT(a.cfg.JWTSecret)
	userRepo := postgres.NewUserRepository(a.dbPool)
	userSvc := service.NewUserService(userRepo, a.redisClient, a.locker)
	userHdl := http.NewUserHandler(*userSvc)
	r := router.InitRouter(userHdl)
	fmt.Printf("系统启动成功，监听端口: %s\n", a.cfg.ServerPort)
	return r.Run(":" + a.cfg.ServerPort)

}
