package sqflite

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/go-flutter-desktop/go-flutter"
	"github.com/go-flutter-desktop/go-flutter/plugin"
	"github.com/go-xorm/xorm"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

const channelName = "com.tekartik.sqflite"

const (
	METHOD_GET_PLATFORM_VERSION = "getPlatformVersion";
	METHOD_GET_DATABASES_PATH   = "getDatabasesPath";
	METHOD_DEBUG_MODE           = "debugMode";
	METHOD_OPTIONS              = "options";
	METHOD_OPEN_DATABASE        = "openDatabase";
	METHOD_CLOSE_DATABASE       = "closeDatabase";
	METHOD_INSERT               = "insert";
	METHOD_EXECUTE              = "execute";
	METHOD_QUERY                = "query";
	METHOD_UPDATE               = "update";
	METHOD_BATCH                = "batch";
	PARAM_ID                    = "id";
	PARAM_PATH                  = "path";
	// when opening a database
	PARAM_READ_ONLY       = "readOnly";       // boolean
	PARAM_SINGLE_INSTANCE = "singleInstance"; // boolean
	// Result when opening a database
	PARAM_RECOVERED         = "recovered";
	PARAM_QUERY_AS_MAP_LIST = "queryAsMapList";        // boolean
	PARAM_THREAD_PRIORITY   = "androidThreadPriority"; // int

	PARAM_SQL               = "sql";
	PARAM_SQL_ARGUMENTS     = "arguments";
	PARAM_NO_RESULT         = "noResult";
	PARAM_CONTINUE_OR_ERROR = "continueOnError";

	// in batch
	PARAM_OPERATIONS = "operations";
	// in each operation
	PARAM_METHOD = "method";

	// Batch operation results
	PARAM_RESULT          = "result";
	PARAM_ERROR           = "error"; // map with code/message/data
	PARAM_ERROR_CODE      = "code";
	PARAM_ERROR_MESSAGE   = "message";
	PARAM_ERROR_DATA      = "data";
	SQLITE_ERROR          = "sqlite_error";    // code
	ERROR_BAD_PARAM       = "bad_param";       // internal only
	ERROR_OPEN_FAILED     = "open_failed";     // msg
	ERROR_DATABASE_CLOSED = "database_closed"; // msg

	// memory database path
	MEMORY_DATABASE_PATH = ":memory:";

	// android log tag
	TAG = "Sqflite";
)

type SqflitePlugin struct {
	sync.Mutex
	VendorName      string
	ApplicationName string

	userConfigFolder string
	codec            plugin.StandardMessageCodec
	databases        map[int]*xorm.Engine
	databaseId       int
}

var _ flutter.Plugin = &SqflitePlugin{} // compile-time type check

func (p *SqflitePlugin) InitPlugin(messenger plugin.BinaryMessenger) error {
	if p.VendorName == "" {
		return errors.New("SqflitePlugin.VendorName must be set")
	}
	if p.ApplicationName == "" {
		return errors.New("SqflitePlugin.ApplicationName must be set")
	}

	switch runtime.GOOS {
	case "darwin":
		home, err := homedir.Dir()
		if err != nil {
			return errors.Wrap(err, "failed to resolve user home dir")
		}
		p.userConfigFolder = filepath.Join(home, "Library", "Application Support")
	case "windows":
		p.userConfigFolder = os.Getenv("APPDATA")
	default:
		// https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
		if os.Getenv("XDG_CONFIG_HOME") != "" {
			p.userConfigFolder = os.Getenv("XDG_CONFIG_HOME")
		} else {
			home, err := homedir.Dir()
			if err != nil {
				return errors.Wrap(err, "failed to resolve user home dir")
			}
			p.userConfigFolder = filepath.Join(home, ".config")
		}
	}

	channel := plugin.NewMethodChannel(messenger, channelName, plugin.StandardMethodCodec{})
	channel.HandleFunc("insert", p.handleInsert)
	channel.HandleFunc("batch", p.handleBatch)
	channel.HandleFunc("debugMode", p.handleDebugMode)
	channel.HandleFunc("options", p.handleOptions)
	channel.HandleFunc("closeDatabase", p.handleCloseDatabase)
	channel.HandleFunc("openDatabase", p.handleOpenDatabase)
	channel.HandleFunc("execute", p.handleExecute)
	channel.HandleFunc("update", p.handleUpdate)
	channel.HandleFunc("query", p.handleQuery)
	channel.HandleFunc("getPlatformVersion", p.handleGetPlatformVersion)
	channel.HandleFunc("getDatabasesPath", p.handleGetDatabasePath)
	channel.HandleFunc("databaseExists", p.handleDatabaseExists)
	//channel.HandleFunc("deleteDatabase", p.handleDeleteDatabase)
	return nil
}

func (p *SqflitePlugin) handleInsert(arguments interface{}) (reply interface{}, err error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	return filepath.Join(cacheDir, p.VendorName, p.ApplicationName), nil
}

func (p *SqflitePlugin) handleBatch(arguments interface{}) (reply interface{}, err error) {
	return filepath.Join(p.userConfigFolder, p.VendorName, p.ApplicationName), nil
}

func (p *SqflitePlugin) handleDebugMode(arguments interface{}) (reply interface{}, err error) {
	return filepath.Join(p.userConfigFolder, p.VendorName, p.ApplicationName), nil
}

func (p *SqflitePlugin) handleOptions(arguments interface{}) (reply interface{}, err error) {
	return filepath.Join(p.userConfigFolder, p.VendorName, p.ApplicationName), nil
}

func (p *SqflitePlugin) handleCloseDatabase(arguments interface{}) (reply interface{}, err error) {
	var db *xorm.Engine
	db, err = p.getDatabase(arguments)
	if err != nil {
		return nil, err
	}
	err = db.Close()
	return nil, err
}

func (p *SqflitePlugin) handleOpenDatabase(arguments interface{}) (reply interface{}, err error) {
	var ok bool
	var args map[string]interface{}
	if args, ok = arguments.(map[string]interface{}); !ok {
		return nil, errors.New("invalid arguments")
	}
	var dbpath string
	if dpath, ok := args[PARAM_PATH]; ok {
		dbpath = dpath.(string)
	}
	if dbpath == "" {
		return nil, errors.New("invalid dbpath")
	}
	var engine *xorm.Engine
	engine, err = xorm.NewEngine("sqlite3", path.Join(p.userConfigFolder, dbpath))
	if err != nil {
		return nil, err
	}
	p.Lock()
	p.databaseId++
	p.databases[p.databaseId] = engine
	p.Unlock()
	return p.databaseId, nil
}

func (p *SqflitePlugin) handleExecute(arguments interface{}) (reply interface{}, err error) {
	return filepath.Join(p.userConfigFolder, p.VendorName, p.ApplicationName), nil
}

func (p *SqflitePlugin) handleUpdate(arguments interface{}) (reply interface{}, err error) {
	return filepath.Join(p.userConfigFolder, p.VendorName, p.ApplicationName), nil
}

func (p *SqflitePlugin) handleQuery(arguments interface{}) (reply interface{}, err error) {
	return filepath.Join(p.userConfigFolder, p.VendorName, p.ApplicationName), nil
}

func (p *SqflitePlugin) handleGetPlatformVersion(arguments interface{}) (reply interface{}, err error) {
	return filepath.Join(p.userConfigFolder, p.VendorName, p.ApplicationName), nil
}

func (p *SqflitePlugin) handleGetDatabasePath(arguments interface{}) (reply interface{}, err error) {
	return filepath.Join(p.userConfigFolder, p.VendorName, p.ApplicationName), nil
}

func (p *SqflitePlugin) handleDatabaseExists(arguments interface{}) (reply interface{}, err error) {
	return filepath.Join(p.userConfigFolder, p.VendorName, p.ApplicationName), nil
}

//func (p *SqflitePlugin) handleDeleteDatabase(arguments interface{}) (reply interface{}, err error) {
//	return filepath.Join(p.userConfigFolder, p.VendorName, p.ApplicationName), nil
//}

func (p *SqflitePlugin) getDatabase(arguments interface{}) (*xorm.Engine, error) {
	switch arguments.(type) {
	case int:
		if db, ok := p.databases[arguments.(int)]; ok {
			return db, nil
		}
	}
	return nil, errors.New("invalid database")
}
