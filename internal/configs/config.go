package configs

import (
	"errors"
	loggerconfs "github.com/apulis/simple-gin-logger/pkg/configs"
	"github.com/spf13/viper"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
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
	InitToolImage    string
	HttpClient  HttpClient
	GatewayUrl  string
	ExtranetAddress string
	ClusterId   string
	VersionControl VCSConfigTable
	EnableNamespace bool
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
	Sslmode      string
}

type ResourceConfig struct{
	Model         string
	Dataset       string
	Jobsched      string
	ApHarbor      string
	ApWorkshop    string
	Code          string
	Iam           string
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
			TimeoutSeconds:10,
		},
	}
	err = viper.Unmarshal(&appConfig)
	if err != nil {
		return nil, err
	}
	//@add: validate root storage data path
	data_path := appConfig.GetStoragePath(appConfig.Storage)
	if len(data_path) == 0 {
		return nil,errors.New("invalid data path mounted !!!")
	}

	if len(appConfig.Time.TimeZoneStr) > 0 {
		appConfig.Time.TimeZone, err = time.LoadLocation(appConfig.Time.TimeZoneStr)
		if err != nil {
			return nil, err
		}
		//@add: set global default timezone
		time.Local=appConfig.Time.TimeZone
	}
	//@add: check log configurations
	if appConfig.Log.WriteFile {
		if len(appConfig.Log.FileDir) == 0 {
			appConfig.Log.FileDir=path.Join(data_path,"_logs_")
		}
		if len(appConfig.Log.FileName) == 0 {
			appConfig.Log.FileName="ai-lab.log"
		}
	}
	//@modify: read postgres password from env

	if pg_passwd , exists := os.LookupEnv("POSTGRES_PASSWORD");exists && !strings.HasPrefix(pg_passwd,"vault:") {
		appConfig.Db.Password=pg_passwd
	}
	//@add: parse gateway url into nodePort address
	if uri , err := url.Parse(appConfig.GatewayUrl);err == nil {
		appConfig.ExtranetAddress=uri.Hostname()
	}
	return appConfig, nil
}

func GetAppConfig() *AppConfig {
	return appConfig
}

func (conf *AppConfig) GetStoragePath(pvcName string)string{
	if strings.HasPrefix(pvcName,"pvc://") {
		pvcName=pvcName[6:]
	}
	return conf.Mounts[pvcName]
}
