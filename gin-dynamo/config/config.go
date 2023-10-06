package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Table                string
	Bucket               string
	AppName              string
	AppEnv               string
	SqlPrefix            string
	RedisAddr            string
	DBUserName           string
	DBPassword           string
	DBHostWriter         string
	DBHostReader         string
	DBPort               string
	DBName               string
	DBMaxOpenConnections int
	DBMaxIdleConnections int
	ServerPort           string
	EsURL                string
	EsPort               int
	SentryDSN            string
	SentrySamplingRate   float64
	AWSAccessKey         string
	AWSSecretKey         string
	JWT_SECRET           string
	KafkaServer          string
	KafkaTopic           string
	Mechanisms           string
	Username             string
	Password             string
	Protocol             string
	Key                  string
	Topic                string
	YoutubeKey           string
	Index                string
	Location             string
	Token                string
}

var config Config
var YoutubeKeyList []string
var YoutubeKey string
var Passcode = 0

// Should run at the very beginning, before any other package
// or code.
func init() {
	appEnv := os.Getenv("APP_ENV")
	fmt.Println(appEnv)
	if len(appEnv) == 0 {
		appEnv = "dev"
	}

	// var e error
	configFilePath := "config/.env"
	if os.Getenv("APP_ENV") == "test" {
		configFilePath = "../config/.env"
	}
	e := godotenv.Load(configFilePath)

	if e != nil {
		fmt.Println("error loading env: ", e)
		panic(e.Error())
	}
	config.AppName = os.Getenv("SERVICE_NAME")
	config.AppEnv = appEnv
	config.SqlPrefix = "/* " + config.AppName + " - " + config.AppEnv + "*/"
	config.RedisAddr = os.Getenv("REDIS_ADDR")
	config.DBUserName = os.Getenv("DB_USERNAME")
	config.DBHostReader = os.Getenv("DB_HOST_READER")
	config.DBHostWriter = os.Getenv("DB_HOST_WRITER")
	config.DBPort = os.Getenv("DB_PORT")
	config.DBPassword = os.Getenv("DB_PASSWORD")
	config.DBName = os.Getenv("DB_NAME")
	config.DBMaxIdleConnections, _ = strconv.Atoi(os.Getenv("DB_MAX_IDLE_CONENCTION"))
	config.DBMaxOpenConnections, _ = strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONNECTIONS"))
	config.ServerPort = os.Getenv("SERVER_PORT")
	config.EsURL = os.Getenv("ES_URL")
	config.EsPort, _ = strconv.Atoi(os.Getenv("ES_PORT"))
	config.SentryDSN = os.Getenv("SENTRY_DSN")
	config.SentrySamplingRate, _ = strconv.ParseFloat(os.Getenv("SENTRY_SAMPLING_RATE"), 64)
	config.AWSAccessKey = os.Getenv("_KEY")
	config.AWSSecretKey = os.Getenv("_SECRET")
	config.JWT_SECRET = os.Getenv("JWT_SECRET")
	config.KafkaServer = os.Getenv("KAFKA_SERVER")
	config.Mechanisms = os.Getenv("sasl.mechanisms")
	config.Username = os.Getenv("sasl.username")
	config.Protocol = os.Getenv("security.protocol")
	config.Password = os.Getenv("sasl.password")
	config.Topic = os.Getenv("topic")
	config.Key = os.Getenv("Key")
	config.Index = os.Getenv("Index")
	config.Bucket = os.Getenv("Bucket")
	config.Table = os.Getenv("Table")
	config.Location = os.Getenv("Location")
	config.Token = os.Getenv("Token")
	YoutubeKey = os.Getenv("YoutubeKey")
	YoutubeKeyList = strings.Split(YoutubeKey, ",")
	UpdateKey()
}

func Get() Config {
	return config
}

func IsProduction() bool {
	return config.AppEnv == "production"
}

func UpdateKey() {
	if Passcode >= len(YoutubeKeyList) {
		Passcode = 0
	}
	config.YoutubeKey = YoutubeKeyList[Passcode]
	Passcode++
}
