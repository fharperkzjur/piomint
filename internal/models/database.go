
package models

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"reflect"

	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/internal/loggers"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB
var logger *logrus.Logger

func InitDb()  error {
	logger = loggers.GetLogger()
	dbConf := configs.GetAppConfig().Db

	var err error

	switch dbConf.ServerType {
	case "postgres", "postgresql":
		// create database if not exists
		preDsn := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=disable", dbConf.Host, dbConf.Port, dbConf.Username, dbConf.Password)
		db, err = gorm.Open(postgres.Open(preDsn), &gorm.Config{})
		if err != nil {
			panic(err)
		}

		exit := 0
		res1 := db.Table("pg_database").Select("count(1)").Where("datname = ?", dbConf.Database).Scan(&exit)
		if res1.Error != nil {
			return res1.Error
		}

		if exit == 0 {
			logger.Info(fmt.Sprintf("Trying to create database: %s", dbConf.Database))
			res2 := db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbConf.Database))
			if res2.Error != nil {
				return res2.Error
			}
		}

		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", dbConf.Host, dbConf.Port, dbConf.Username, dbConf.Password, dbConf.Database)
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			return err
		}
	default:
		return errors.New("Unsupported database type")
	}
	if configs.GetAppConfig().Db.Debug{
		db = db.Debug()
	}
	//@add: create database for gitea
	db.Exec("create database gitea")

	return  initTables()
}

func initTables() error {

	modelTypes := []interface{}{
		&Lab{},
		&Run{},
		&Event{},
		&System{},
		&Link{},
	}

	for _, modelType := range modelTypes {
		err := autoMigrateTable(modelType)
		if err != nil {
			return err
		}
	}
	//@mark:  check db version compatible, otherwise will exit service
	CheckAiLabsDBVersion()
	return nil
}

func autoMigrateTable(modelType interface{}) error {
	val := reflect.Indirect(reflect.ValueOf(modelType))
	modelName := val.Type().Name()

	logger.Infof("Migrating Table of %s ...", modelName)

	err := db.AutoMigrate(modelType)
	if err != nil {
		return err
	}
	return nil
}
