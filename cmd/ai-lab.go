package main

import (
	"context"
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/internal/grpc_server"
	"github.com/apulis/bmod/ai-lab-backend/internal/loggers"
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/internal/routers"
	"github.com/apulis/bmod/ai-lab-backend/internal/services"
	pb "github.com/apulis/bmod/ai-lab-backend/pkg/api"
	_ "github.com/apulis/sdk/go-utils/logging"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var logger * logrus.Logger

func main() {
	config, err := configs.InitConfig()
	if err != nil {
		panic(err)
	}
	logger, err = loggers.InitLogger(&config.Log)
	if err != nil {
		panic(err)
	}
	err = models.InitDb()
	if err != nil {
		panic(err)
	}
	err = services.InitServices()
	if err != nil {
		panic(err)
	}
	var httpSrv,grpcSrv interface{}

	if config.Port != 0 {
		httpSrv=startHttpServer(config.Port)
	}
	if config.Grpc != 0 {
		grpcSrv=startGrpcServer(config.Grpc)
	}
	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("Shutting down server...")

	if httpSrv != nil{
		stopHttpServer(httpSrv,5*time.Second)
		httpSrv = nil
	}
	if grpcSrv != nil {
		stopGrpcServer(grpcSrv,5*time.Second)
		grpcSrv = nil
	}
	logger.Println("Server exited !")

	services.QuitServices()
	logger.Println("Server backend tasks exited !")
}

func startHttpServer(port int ) *http.Server{
	//@mark: initialize http web server and start
	router := routers.InitRouter()
	logger.Info("Application started, listening and serving HTTP on: ", port)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("listen: %s\n", err)
		}
	}()
	return srv
}
func startGrpcServer(port int ) interface{} {
	 tcp, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	 if err != nil {
		log.Fatalf("failed to listen: %v", err)
	 }
	 srv := grpc.NewServer()
	 logger.Info("Application started, listening and serving grpc on: ", port)
	 pb.RegisterAILabServer(srv,&grpc_server.AILabServerImpl{})

	 go func() {
		 srv.Serve(tcp)
	 }()

     return srv
}

func stopHttpServer(server interface{},ts time.Duration) {
	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	srv := server.(*http.Server)
	ctx, cancel := context.WithTimeout(context.Background(), ts)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown:", err)
	}
}
func stopGrpcServer(server interface{},ts time.Duration) {
	srv := server.(*grpc.Server)
	srv.Stop()
	//srv.GracefulStop()
}
/*
func startJobMonitor(config configs.RabbitmqConfig) interface{}{
	addr := fmt.Sprintf("amqp://%s:%s@%s:%d",config.User,config.Password,config.Host,config.Port)
	b := rabbitmq.NewBroker(
		broker.Addrs(addr),
		rabbitmq.ExchangeName("default"),
	)
	if err := b.Connect(); err != nil {
		logger.Fatalf("connect rabbitmq:%s error:%s !",addr,err.Error())
	}
	go func() {
		_, err := b.Subscribe(fmt.Sprintf("%v",exports.AILAB_MODULE_ID), services.MonitorJobStatus, rabbitmq.DurableQueue())
		if err != nil {
			logger.Fatalf("Subscribe rabbitmq:%s error:%s !",addr,err.Error())
		}
	}()
	return b
}
func stopJobMonitor(listen interface{},ts time.Duration){
     broker := listen.(broker.Broker)
     broker.Disconnect()
}*/
