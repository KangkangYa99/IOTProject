package main

import (
	"IOTProject/internal/app"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	application, cleanup, err := app.New()
	if err != nil {
		log.Fatalf("系统初始化失败: %v", err)
	}
	defer cleanup()
	go func() {
		if err = application.Run(); err != nil {
			log.Fatalf("系统运行错误: %v", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🌬️  正在关闭服务器...")
}
