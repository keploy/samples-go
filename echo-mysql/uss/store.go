package uss

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var MetaStore *Store

type ShortCodeInfo struct {
	UID       uint64    `json:"id" sql:"AUTO_INCREMENT" gorm:"primary_key"`
	ShortCode string    `json:"shortcode" gorm:"uniqueIndex"`
	URL       string    `json:"url"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime(6);autoUpdateTime"`
	EndTime   time.Time `json:"end_time"   gorm:"type:datetime(6)"`
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

// Upsert by unique short_code to avoid conflicts on reseed.
func (s *Store) UpsertByShortCode(info *ShortCodeInfo) error {
	info.EndTime = ToDBLocalMicro(info.EndTime)
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "short_code"}},
		DoUpdates: clause.AssignmentColumns([]string{"url", "updated_at", "end_time", "created_by"}),
	}).Create(info).Error
}

func (s *Store) UpsertMany(infos []*ShortCodeInfo) error {
	for _, i := range infos {
		if err := s.UpsertByShortCode(i); err != nil {
			return err
		}
	}
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

// Exact match on end_time (normalized to µs). NOTE: uses "=" (not BETWEEN).
func (s *Store) FindByEndTime(t time.Time) []ShortCodeInfo {
	t = ToDBLocalMicro(t) // match what we store
	var infos []ShortCodeInfo
	s.db.Where("end_time = ?", t).Order("short_code asc").Find(&infos)
	return infos
}

// Sentinel helpers
var (
	SentinelStart = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
	SentinelMax   = time.Date(9999, 12, 31, 23, 59, 59, 999999000, time.UTC) // .999999
)

func (s *Store) FindSentinels() []ShortCodeInfo {
	ss := ToDBLocalMicro(SentinelStart)
	sm := ToDBLocalMicro(SentinelMax)
	var infos []ShortCodeInfo
	s.db.Where("end_time IN (?, ?)", ss, sm).Order("end_time asc, short_code asc").Find(&infos)
	return infos
}

// For the demo set (CreatedBy == "keploy.io/dates")
func (s *Store) FindSeededDates() []ShortCodeInfo {
	var infos []ShortCodeInfo
	s.db.Where("created_by = ?", "keploy.io/dates").Order("end_time asc, short_code asc").Find(&infos)
	return infos
}

// FindActive remains as-is
func (s *Store) FindActive() []ShortCodeInfo {
	var infos []ShortCodeInfo
	s.db.Where("end_time > ?", time.Now()).Find(&infos)
	return infos
}
