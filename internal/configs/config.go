package configs

import (
	"time"

	loggerconfs "github.com/apulis/simple-gin-logger/pkg/configs"
	"github.com/spf13/viper"
)

var appConfig *AppConfig

type AppConfig struct {
	Port        int
	Grpc        int
	Log         loggerconfs.LogConfig
	ApiV1Prefix string
	Db          DbConfig
	Time        TimeConfig
	Resources   ResourceConfig
	Rabbitmq    RabbitmqConfig
	Debug       bool
	Mounts      map[string]string
	Storage     string
	HttpClient  HttpClient
}

type DbConfig struct {
	ServerType   string
	Username     string
	Password     string
	Host         string
	Port         int
	Database     string
	MaxOpenConns int
	MaxIdleConns int
	Debug        bool
}

type ResourceConfig struct{
	Model         string
	Dataset       string
	Jobsched      string
	Engine        string
	Code          string
}

type TimeConfig struct {
	TimeZoneStr string
	TimeZone    *time.Location
}

type RabbitmqConfig struct{
	User      string
	Password  string
	Host      string
	Port      int
}

type HttpClient struct {
	MaxIdleConns        int
	MaxConnsPerHost     int
	MaxIdleConnsPerHost int
	TimeoutSeconds      int
}

func InitConfig() (*AppConfig, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath("configs")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	appConfig = &AppConfig{
		HttpClient:  HttpClient{
			MaxIdleConns :100,
			MaxConnsPerHost:100,
			MaxIdleConnsPerHost:100,
			TimeoutSeconds:5,
		},
	}
	err = viper.Unmarshal(&appConfig)
	if err != nil {
		return nil, err
	}

	if appConfig.Time.TimeZoneStr == "" {
		appConfig.Time.TimeZoneStr = "Asia/Shanghai"
	}
	appConfig.Time.TimeZone, err = time.LoadLocation(appConfig.Time.TimeZoneStr)
	if err != nil {
		return nil, err
	}

	return appConfig, nil
}

func GetAppConfig() *AppConfig {
	return appConfig
}

