package dao

import (
	"errors"
	"fmt"
	"h-ui/model/constant"
	"log"
	"os"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var sqlInitStr = `
CREATE TABLE IF NOT EXISTS account (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    username       TEXT    NOT NULL UNIQUE DEFAULT '',
    pass           TEXT    NOT NULL        DEFAULT '',
    con_pass       TEXT    NOT NULL        DEFAULT '',
    quota          INTEGER NOT NULL        DEFAULT 0,
    download       INTEGER NOT NULL        DEFAULT 0,
    upload         INTEGER NOT NULL        DEFAULT 0,
    expire_time    INTEGER NOT NULL        DEFAULT 0,
    kick_util_time INTEGER NOT NULL        DEFAULT 0,
    device_no      INTEGER NOT NULL        DEFAULT 3,
    role           TEXT    NOT NULL        DEFAULT 'user',
    deleted        INTEGER NOT NULL        DEFAULT 0,
    create_time    TIMESTAMP               DEFAULT CURRENT_TIMESTAMP,
    update_time    TIMESTAMP               DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE account ADD COLUMN login_at INTEGER NOT NULL DEFAULT 0;
ALTER TABLE account ADD COLUMN con_at INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS account_deleted_index ON account (deleted);
CREATE INDEX IF NOT EXISTS account_username_index ON account (username);
CREATE INDEX IF NOT EXISTS account_con_pass_index ON account (con_pass);
CREATE INDEX IF NOT EXISTS account_pass_index ON account (pass);

INSERT INTO account (id, username, pass, con_pass, quota, download, upload, expire_time, device_no, role)
SELECT 1, 'sysadmin', '02f382b76ca1ab7aa06ab03345c7712fd5b971fb0c0f2aef98bac9cd', 'sysadmin.sysadmin', 
       -1, 0, 0, 253370736000000, 6, 'admin'
WHERE NOT EXISTS (SELECT 1 FROM account WHERE id = 1);

CREATE TABLE IF NOT EXISTS config (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    key         TEXT NOT NULL UNIQUE DEFAULT '',
    value       TEXT NOT NULL        DEFAULT '',
    remark      TEXT NOT NULL        DEFAULT '',
    create_time TIMESTAMP            DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMP            DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS config_key_index ON config (key);

INSERT INTO config (key, value, remark)
SELECT 'H_UI_WEB_PORT', '8081', 'H UI Web Port'
WHERE NOT EXISTS (SELECT 1 FROM config WHERE key = 'H_UI_WEB_PORT');

INSERT INTO config (key, value, remark)
SELECT 'H_UI_WEB_CONTEXT', '/', 'H UI Web Context'
WHERE NOT EXISTS (SELECT 1 FROM config WHERE key = 'H_UI_WEB_CONTEXT');

INSERT INTO config (key, value, remark)
SELECT 'H_UI_CRT_PATH', '', 'H UI Crt File Path'
WHERE NOT EXISTS (SELECT 1 FROM config WHERE key = 'H_UI_CRT_PATH');

INSERT INTO config (key, value, remark)
SELECT 'H_UI_KEY_PATH', '', 'H UI Key File Path'
WHERE NOT EXISTS (SELECT 1 FROM config WHERE key = 'H_UI_KEY_PATH');

INSERT INTO config (key, value, remark)
SELECT 'JWT_SECRET', hex(randomblob(10)), 'JWT Secret'
WHERE NOT EXISTS (SELECT 1 FROM config WHERE key = 'JWT_SECRET');

INSERT INTO config (key, value, remark)
SELECT 'HYSTERIA2_ENABLE', '0', 'Hysteria2 Switch'
WHERE NOT EXISTS (SELECT 1 FROM config WHERE key = 'HYSTERIA2_ENABLE');

INSERT INTO config (key, value, remark)
SELECT 'HYSTERIA2_CONFIG', '', 'Hysteria2 Config'
WHERE NOT EXISTS (SELECT 1 FROM config WHERE key = 'HYSTERIA2_CONFIG');

INSERT INTO config (key, value, remark)
SELECT 'HYSTERIA2_TRAFFIC_TIME', '1', 'Hysteria2 Traffic Time'
WHERE NOT EXISTS (SELECT 1 FROM config WHERE key = 'HYSTERIA2_TRAFFIC_TIME');

INSERT INTO config (key, value, remark)
SELECT 'HYSTERIA2_CONFIG_REMARK', '', 'Hysteria2 Config Remark'
WHERE NOT EXISTS (SELECT 1 FROM config WHERE key = 'HYSTERIA2_CONFIG_REMARK');

INSERT INTO config (key, value, remark)
SELECT 'HYSTERIA2_CONFIG_PORT_HOPPING', '', 'Hysteria2 Config Port Hopping'
WHERE NOT EXISTS (SELECT 1 FROM config WHERE key = 'HYSTERIA2_CONFIG_PORT_HOPPING');

INSERT INTO config (key, value, remark)
SELECT 'RESET_TRAFFIC_CRON', '', 'Reset Traffic Cron'
WHERE NOT EXISTS (SELECT 1 FROM config WHERE key = 'RESET_TRAFFIC_CRON');

INSERT INTO config (key, value, remark)
SELECT 'TELEGRAM_ENABLE', '0', 'Telegram Switch'
WHERE NOT EXISTS (SELECT 1 FROM config WHERE key = 'TELEGRAM_ENABLE');

INSERT INTO config (key, value, remark)
SELECT 'TELEGRAM_TOKEN', '', 'Telegram Token'
WHERE NOT EXISTS (SELECT 1 FROM config WHERE key = 'TELEGRAM_TOKEN');

INSERT INTO config (key, value, remark)
SELECT 'TELEGRAM_CHAT_ID', '', 'Telegram ChatId'
WHERE NOT EXISTS (SELECT 1 FROM config WHERE key = 'TELEGRAM_CHAT_ID');

INSERT INTO config (key, value, remark)
SELECT 'TELEGRAM_LOGIN_JOB_ENABLE', '0', 'TELEGRAM LOGIN Notification'
WHERE NOT EXISTS (SELECT 1 FROM config WHERE key = 'TELEGRAM_LOGIN_JOB_ENABLE');

INSERT INTO config (key, value, remark)
SELECT 'TELEGRAM_LOGIN_JOB_TEXT', '[time], [username] logged into the panel, IP address is [ip]', 'TELEGRAM LOGIN Notification Text'
WHERE NOT EXISTS (SELECT 1 FROM config WHERE key = 'TELEGRAM_LOGIN_JOB_TEXT');

INSERT INTO config (key, value, remark)
SELECT 'CLASH_EXTENSION', '', 'Clash Subscription Extension'
WHERE NOT EXISTS (SELECT 1 FROM config WHERE key = 'CLASH_EXTENSION');
`

var sqliteDB *gorm.DB

func InitSqliteDB() error {
	var err error
	sqliteDB, err = gorm.Open(sqlite.Open(fmt.Sprintf("%s%s", os.Getenv("HUI_DATA"), constant.SqliteDBPath)), &gorm.Config{
		TranslateError: true,
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Silent,
				IgnoreRecordNotFoundError: true,
				ParameterizedQueries:      true,
				Colorful:                  false,
			},
		),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		logrus.Errorf("sqlite open err: %v", err)
		return errors.New("sqlite open err")
	}
	return nil
}

func InitSql(port string) error {
	if err := InitSqliteDB(); err != nil {
		return err
	}
	if err := sqliteInit(sqlInitStr); err != nil {
		return err
	}
	if port != "" {
		var result string
		db, err := sqliteDB.DB()
		if err != nil {
			return err
		}
		if err := db.QueryRow("SELECT value from config where key = 'H_UI_WEB_PORT' limit 1").Scan(&result); err != nil {
			logrus.Errorf("sqlite exec err: %v", err)
			return errors.New("sqlite exec err")
		}

		if result == "8081" {
			if tx := sqliteDB.Exec("UPDATE config set value = ? where key = 'H_UI_WEB_PORT'", port); tx.Error != nil {
				logrus.Errorf("sqlite exec err: %v", tx.Error)
				return errors.New("sqlite exec err")
			}
		}
	}
	return nil
}

func sqliteInit(sqlStr string) error {
	if sqliteDB != nil {
		sqls := strings.Split(strings.Replace(sqlStr, "\r\n", "\n", -1), ";\n")
		for _, s := range sqls {
			s = strings.TrimSpace(s)
			if s != "" {
				tx := sqliteDB.Exec(s)
				if tx.Error != nil && !strings.HasPrefix(tx.Error.Error(), "SQL logic error: duplicate column name") {
					logrus.Errorf("sqlite exec err: %v", tx.Error)
					return errors.New("sqlite exec err")
				}
			}
		}
	}
	return nil
}

func CloseSqliteDB() error {
	if sqliteDB != nil {
		db, err := sqliteDB.DB()
		if err != nil {
			logrus.Errorf("sqlite err: %v", err)
			return errors.New("sqlite err")
		}
		if err = db.Close(); err != nil {
			logrus.Errorf("sqlite close err: %v", err)
			return errors.New("sqlite close err")
		}
	}
	return nil
}

func Paginate(pageNum *int64, pageSize *int64) func(db *gorm.DB) *gorm.DB {
	var num int64 = 1
	var size int64 = 10
	if pageNum != nil && *pageNum > 0 {
		num = *pageNum
	}
	if pageSize != nil && *pageSize > 0 {
		size = *pageSize
	}
	return func(db *gorm.DB) *gorm.DB {
		return db.Offset(int((num - 1) * size)).Limit(int(size))
	}
}
