package config

import (
	"os"
	"strconv"
)

type Config struct {
	AppName              string
	DATABASE             string
	AppEnv               string
	SqlPrefix            string
	DBUserName           string
	PORT                 string
	DBPassword           string
	DBHostWriter         string
	ServerPort           string
	DBHostReader         string
	DBPort               string
	DBName               string
	AMPQ_URL             string
	DAILYPASSSERVICE     string
	DBMaxOpenConnections int
	EmailPassword        string
	DBMaxIdleConnections int
	JWT                  string
	MongoPort            string
	MongoUsername        string
	MongoPassword        string
	CloudName            string
	CloudSecret          string
	CloudPublic          string
	SuperAdmin           string
}

var config Config

// Should run at the very beginning, before any other package
// or code.
func init() {
	appEnv := os.Getenv("APP_ENV")
	if len(appEnv) == 0 {
		appEnv = "dev"
	}
	config.AppName = os.Getenv("SERVICE_NAME")
	config.AppEnv = appEnv
	config.SqlPrefix = "/* " + config.AppName + " - " + config.AppEnv + "*/"
	config.DBUserName = os.Getenv("DB_USERNAME")
	config.JWT = os.Getenv("JWT_TOKEN")
	config.DBHostWriter = os.Getenv("DB_HOST_WRITER")
	config.DBPort = os.Getenv("DB_PORT")
	config.DBPassword = os.Getenv("DB_PASSWORD")
	config.DBName = os.Getenv("DB_NAME")
	config.DATABASE = os.Getenv("DATABASE")
	config.AMPQ_URL = os.Getenv("AMPQ_URL")
	config.PORT = os.Getenv("PORT")
	config.DAILYPASSSERVICE = os.Getenv("DAILYPASSSERVICE")
	config.EmailPassword = os.Getenv("EMAIL_PASSWORD")
	config.DBMaxIdleConnections, _ = strconv.Atoi(os.Getenv("DB_MAX_IDLE_CONENCTION"))
	config.DBMaxOpenConnections, _ = strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONNECTIONS"))
	config.DBHostReader = os.Getenv("DB_HOST_READER")
	config.MongoPort = os.Getenv("MONGO_PORT")
	config.MongoUsername = os.Getenv("MongoUsername")
	config.MongoPassword = os.Getenv("MongoPassword")
	config.CloudSecret = os.Getenv("CloudSecret")
	config.CloudName = os.Getenv("CloudName")
	config.CloudPublic = os.Getenv("CloudPublic")
	config.SuperAdmin = os.Getenv("SuperAdmin")
	config.ServerPort = os.Getenv("PORT")
}

func Get() Config {
	return config
}
