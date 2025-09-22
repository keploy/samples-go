package uss

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var MetaStore *Store

type ShortCodeInfo struct {
	UID       uint64 `json:"id" sql:"AUTO_INCREMENT" gorm:"primary_key"`
	ShortCode string `json:"shortcode" gorm:"uniqueIndex"`
	URL       string `json:"url"`

	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime(6);autoUpdateTime"`
	EndTime   time.Time `json:"end_time" gorm:"type:datetime(6)"`
	CreatedBy string    `json:"created_by"`
}

type Store struct {
	db *gorm.DB
}

func (s *Store) Connect(config map[string]string) error {
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
		log.Printf("%s", fmt.Sprintf("Failed to create/update db tables with error %s", err.Error()))
		os.Exit(1)
	}

	return nil
}

func (s *Store) Close() {
	db, _ := s.db.DB()
	if err := db.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Could not close database connection: %v\n", err)
		os.Exit(1)
	}
}

func (s *Store) Persist(info *ShortCodeInfo) error {
	s.db.Save(info)
	return nil
}

func (s *Store) FindByShortCode(shortCode string) *ShortCodeInfo {
	var infos []ShortCodeInfo
	s.db.Order("updated_at desc").Find(&infos, "short_code = ?", shortCode)
	if len(infos) == 0 {
		return nil
	}

	urlInfo := infos[0]
	return &urlInfo
}

// FindActive retrieves all records that have not yet expired.
func (s *Store) FindActive() []ShortCodeInfo {
	var infos []ShortCodeInfo
	// Query for records where the end_time is greater than the current time.
	s.db.Where("end_time > ?", time.Now()).Find(&infos)
	return infos
}
