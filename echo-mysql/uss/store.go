package uss

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type ShortCodeInfo struct {
	UID       uint64    `json:"id" sql:"AUTO_INCREMENT" gorm:"primary_key"`
	ShortCode string    `json:"shortcode" gorm:"uniqueIndex"`
	URL       string    `json:"url"`
	UpdatedAt time.Time `json:"updated_at" gorm:"datetime(0);autoUpdateTime"`
}

type USSStore struct {
	db *gorm.DB
}

func (s *USSStore) Connect(config map[string]string) error {
	// Open up our database connection.
	var err error
	mysqlDSN := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local&tls=False",
		config["MYSQL_USER"],
		config["MYSQL_PASSWORD"],
		config["MYSQL_HOST"],
		config["MYSQL_PORT"],
		config["MYSQL_DBNAME"],
	)
	s.db, err = gorm.Open(mysql.New(mysql.Config{
		DSN:               mysqlDSN,
		DefaultStringSize: 256,
	}), &gorm.Config{})
	if err != nil {
		return err
	}

	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}

	sqlDB.SetConnMaxLifetime(1 * time.Hour)
	sqlDB.SetMaxIdleConns(100)
	sqlDB.SetMaxOpenConns(512)

	if err = s.db.AutoMigrate(&ShortCodeInfo{}); err != nil {
		log.Fatal(fmt.Sprintf("Failed to create/update db tables with error %s", err.Error()))
	}

	return nil
}

func (s *USSStore) Close() {
	db, _ := s.db.DB()
	db.Close()
}

func (s *USSStore) Persist(info *ShortCodeInfo) error {
	s.db.Save(info)
	return nil
}

func (s *USSStore) FindByShortCode(shortCode string) *ShortCodeInfo {
	var infos []ShortCodeInfo
	s.db.Order("updated_at desc").Find(&infos, "short_code = ?", shortCode)
	if len(infos) == 0 {
		return nil
	} else {
		urlInfo := infos[0]
		return &urlInfo
	}
}

var MetaStore *USSStore
