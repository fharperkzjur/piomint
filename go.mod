module github.com/apulis/bmod/ai-lab-backend

go 1.16

require (
	github.com/apulis/go-business v0.0.0
	github.com/apulis/sdk/go-utils v0.0.0
	github.com/apulis/simple-gin-logger v0.0.0
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/gin-contrib/cors v1.3.1
	github.com/gin-contrib/sessions v0.0.3
	github.com/gin-gonic/gin v1.7.2
	github.com/google/uuid v1.2.0
	github.com/jackc/pgproto3/v2 v2.0.7 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/pelletier/go-toml v1.9.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/securego/gosec/v2 v2.8.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.7.1
	github.com/swaggo/gin-swagger v1.3.0
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	golang.org/x/sys v0.0.0-20210608053332-aa57babbf139 // indirect
	golang.org/x/text v0.3.6 // indirect
	golang.org/x/tools v0.1.2 // indirect
	google.golang.org/grpc v1.38.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.1.0 // indirect
	google.golang.org/protobuf v1.26.0
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gorm.io/datatypes v1.0.1 // indirect
	gorm.io/driver/postgres v1.1.0
	gorm.io/gorm v1.21.9
	gorm.io/plugin/soft_delete v1.0.1
	k8s.io/apimachinery v0.21.1
)

replace (
	github.com/apulis/go-business => ./deps/go-business
	github.com/apulis/sdk/go-utils => ./deps/go-utils
	github.com/apulis/simple-gin-logger => ./deps/simple-gin-logger
)
