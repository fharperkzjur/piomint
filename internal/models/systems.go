package models

import (
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"log"
)

const (
	sys_ai_lab_db_version= "$ai_lab_ver$"
)

const (
	ai_lab_db_ver = "1.0" // change this value when upgrade from old db
)

type System struct{
	Name    string `gorm:"primary_key" json:"name"`
	Value   string
}

func SetConfigValue(key , value string) APIError {
	return wrapDBUpdateError(db.Where("name=?",key).
		Assign(System{Value:value,Name:key}).
		FirstOrCreate(&System{}),1)
}

func GetConfigValue(key string) (string,APIError) {
	config := &System{Name:key}
	err :=  wrapDBQueryError(db.First(config))
	if err != nil {
		return "",err
	}
	return config.Value,nil
}

func CheckAiLabsDBVersion() APIError{
	ver,err := GetConfigValue(sys_ai_lab_db_version)
	if err == nil {
		if ver != ai_lab_db_ver {
			log.Fatalf("ai_lab current db version: %s , does not match exists db version:%s , may need upgrade !!!",
				ai_lab_db_ver,ver)
		}
	}else if err.Errno() == exports.AILAB_NOT_FOUND{
		err = SetConfigValue(sys_ai_lab_db_version,ai_lab_db_ver)
		if err != nil {
			log.Fatalf("ai_lab initialize version settings error %v !!!",err)
			return err
		}

	}else{
		log.Fatalf("ai_lab initialize error , cannot read db version %v !!!",err)
	}
	return err
}

