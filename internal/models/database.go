/* ******************************************************************************
* 2019 - present Contributed by Apulis Technology (Shenzhen) Co. LTD
*
* This program and the accompanying materials are made available under the
* terms of the MIT License, which is available at
* https://www.opensource.org/licenses/MIT
*
* See the NOTICE file distributed with this work for additional
* information regarding copyright ownership.
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
* WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
* License for the specific language governing permissions and limitations
* under the License.
*
* SPDX-License-Identifier: MIT
******************************************************************************/
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
		preDsn := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=%s", dbConf.Host, dbConf.Port, dbConf.Username, dbConf.Password,dbConf.Sslmode)
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

		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", dbConf.Host, dbConf.Port, dbConf.Username, dbConf.Password, dbConf.Database,dbConf.Sslmode)
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

func OpenDB(driver string,host string,port int,user ,passwd string,database string,sslmode string) (db*gorm.DB,err error){

	switch driver {
	case "postgres", "postgresql":
		// create database if not exists
		preDsn := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=%s",
			     host, port, user, passwd,sslmode)
		db, err = gorm.Open(postgres.Open(preDsn), &gorm.Config{})
		if err != nil {
			panic(err)
		}

		exit := 0
		res1 := db.Table("pg_database").Select("count(1)").Where("datname = ?", database).Scan(&exit)
		if res1.Error != nil {
			return nil,res1.Error
		}
		if exit == 0 {
			res2 := db.Exec(fmt.Sprintf("CREATE DATABASE %s", database))
			if res2.Error != nil {
				return nil,res2.Error
			}
		}

		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			 host, port, user, passwd, database,sslmode)
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			return nil,err
		}
	default:
		return nil,errors.New("Unsupported database type")
	}
	return
}

func initTables() error {

	modelTypes := []interface{}{
		&Lab{},
		&Run{},
		&Event{},
		&System{},
		&Link{},
		&Code{},
		&CodeBind{},
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
