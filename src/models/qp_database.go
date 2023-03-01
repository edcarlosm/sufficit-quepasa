package models

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	migrate "github.com/joncalhoun/migrate"
	log "github.com/sirupsen/logrus"

	"path/filepath"
	"runtime"
	"strconv"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type QpDatabase struct {
	Config     QpDatabaseConfig
	Connection *sqlx.DB
	Users      QpDataUsersInterface
	Servers    QpDataServersInterface
	Webhooks   QpDataWebhooksInterface
}

var (
	Sync       sync.Once // Objeto de sinaleiro para garantir uma única chamada em todo o andamento do programa
	Connection *sqlx.DB
)

// GetDB returns a database connection for the given
// database environment variables
func GetDB() *sqlx.DB {
	Sync.Do(func() {
		config := GetDBConfig()

		// Tenta realizar a conexão
		dbconn, err := sqlx.Connect(config.Driver, config.GetConnectionString())
		if err != nil {
			log.Println(err)
		}

		dbconn.DB.SetMaxIdleConns(500)
		dbconn.DB.SetMaxOpenConns(1000)
		dbconn.DB.SetConnMaxLifetime(30 * time.Second)

		if err != nil {
			log.Println(err)
		}

		// Definindo uma única conexão para todo o sistema
		Connection = dbconn
	})
	return Connection
}

func GetDatabase() *QpDatabase {
	db := GetDB()
	config := GetDBConfig()
	var iusers = QpDataUserSql{db}
	var iwebhooks = QpDataServerWebhookSql{db}
	var iservers = QpDataServerSql{db}

	return &QpDatabase{
		config,
		db,
		iusers,
		iservers,
		iwebhooks}
}

func GetDBConfig() QpDatabaseConfig {
	config := QpDatabaseConfig{}

	config.Driver = os.Getenv("DBDRIVER")
	if len(config.Driver) == 0 {
		config.Driver = "sqlite3"
	}

	config.Host = os.Getenv("DBHOST")
	config.DataBase = os.Getenv("DBDATABASE")
	config.Port = os.Getenv("DBPORT")
	config.User = os.Getenv("DBUSER")
	config.Password = os.Getenv("DBPASSWORD")
	config.SSL = os.Getenv("DBSSLMODE")
	return config
}

// MigrateToLatest updates the database to the latest schema
func MigrateToLatest() (err error) {
	strMigrations := os.Getenv("MIGRATIONS")
	if len(strMigrations) == 0 {
		return
	}

	var fullPath string
	boolMigrations, err := strconv.ParseBool(strMigrations)
	if err == nil {
		// Caso false, migrações não habilitadas
		// Retorna sem problemas
		if !boolMigrations {
			return
		}
	} else {
		fullPath = strMigrations
	}

	log.Info("Migrating database (if necessary)")
	if boolMigrations {
		workDir, err := os.Getwd()
		if err != nil {
			return err
		}

		if runtime.GOOS == "windows" {
			log.Debug("Migrating database on Windows")

			// windows ===================
			leadingWindowsUnit, _ := filepath.Rel("z:\\", workDir)
			migrationsDir := filepath.Join(leadingWindowsUnit, "migrations")
			fullPath = fmt.Sprintf("/%s", strings.ReplaceAll(migrationsDir, "\\", "/"))
		} else {
			// linux ===================
			migrationsDir := filepath.Join(workDir, "migrations")
			fullPath = fmt.Sprintf("file://%s", strings.TrimLeft(migrationsDir, "/"))
		}
	}

	log.Debugf("fullpath database: %s", fullPath)

	config := GetDBConfig()
	superDB := *GetDB()
	db := superDB.DB

	migrator := migrate.Sqlx{
		Printf: func(format string, args ...interface{}) (int, error) {
			log.Println(format, args)
			return 0, nil
		},
		Migrations: Migrations(fullPath),
	}

	log.Debug("Migrating ...")
	err = migrator.Migrate(db, config.Driver)
	if err != nil {
		log.Fatal(err)
	}

	log.Debug("Migrating finished")
	return nil
}

func Migrations(fullPath string) (migrations []migrate.SqlxMigration) {
	log.Debugf("Migrating files from: %s", fullPath)
	files, err := ioutil.ReadDir(fullPath)
	if err != nil {
		log.Fatal(err)
	}

	log.Debug("Migrating creating array with definitions")
	confMap := make(map[string]*QPMigrationFile)

	for _, file := range files {
		info := file.Name()
		dotSplitted := strings.Split(info, ".")      // file name splitted by dots
		extension := dotSplitted[len(dotSplitted)-1] // file extension
		if extension == "sql" {
			id := strings.Split(info, "_")[0]

			title := strings.TrimPrefix(dotSplitted[0], id+"_")
			status := dotSplitted[1]
			filepath := fullPath + "/" + info
			if v, ok := confMap[id]; ok {
				if status == "up" {
					v.FileUp = filepath
				} else if status == "down" {
					v.FileDown = filepath
				}
			} else {
				if status == "up" {
					confMap[id] = &QPMigrationFile{id, title, filepath, ""}
				} else if status == "down" {
					confMap[id] = &QPMigrationFile{id, title, "", filepath}
				}
			}
		}
	}

	for _, migration := range confMap {
		migrations = append(migrations, migrate.SqlxFileMigration(migration.ID, migration.FileUp, migration.FileDown))
	}

	return
}
