// Package sqflite provides an implementation of flutter sqflite plugin for desktop.
//
// Extra dependencies:
//   github.com/go-flutter-desktop/go-flutter
//   github.com/go-flutter-desktop/go-flutter/plugin
//	 github.com/mattn/go-sqlite3
//   github.com/mitchellh/go-homedir
//   github.com/pkg/errors

package sqflite

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/go-flutter-desktop/go-flutter"
	"github.com/go-flutter-desktop/go-flutter/plugin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

const channelName = "com.tekartik.sqflite"

const errorFormat = "[SQFLITE] %v\n"

const (
	METHOD_GET_PLATFORM_VERSION = "getPlatformVersion"
	METHOD_GET_DATABASES_PATH   = "getDatabasesPath"
	METHOD_DEBUG_MODE           = "debugMode"
	METHOD_OPTIONS              = "options"
	METHOD_OPEN_DATABASE        = "openDatabase"
	METHOD_CLOSE_DATABASE       = "closeDatabase"
	METHOD_INSERT               = "insert"
	METHOD_EXECUTE              = "execute"
	METHOD_QUERY                = "query"
	METHOD_UPDATE               = "update"
	METHOD_BATCH                = "batch"
	PARAM_ID                    = "id"
	PARAM_PATH                  = "path"
	// when opening a database
	PARAM_READ_ONLY       = "readOnly"       // boolean
	PARAM_SINGLE_INSTANCE = "singleInstance" // boolean
	// Result when opening a database
	PARAM_RECOVERED         = "recovered"
	PARAM_QUERY_AS_MAP_LIST = "queryAsMapList" // boolean

	PARAM_SQL               = "sql"
	PARAM_SQL_ARGUMENTS     = "arguments"
	PARAM_NO_RESULT         = "noResult"
	PARAM_CONTINUE_OR_ERROR = "continueOnError"

	// in batch
	PARAM_OPERATIONS = "operations"
	// in each operation
	PARAM_METHOD = "method"

	// Batch operation results
	PARAM_RESULT          = "result"
	PARAM_ERROR           = "error" // map with code/message/data
	PARAM_ERROR_CODE      = "code"
	PARAM_ERROR_MESSAGE   = "message"
	PARAM_ERROR_DATA      = "data"
	SQLITE_ERROR          = "sqlite_error"    // code
	ERROR_BAD_PARAM       = "bad_param"       // internal only
	ERROR_OPEN_FAILED     = "open_failed"     // msg
	ERROR_DATABASE_CLOSED = "database_closed" // msg

	// memory database path
	MEMORY_DATABASE_PATH = ":memory:"
)

type SqflitePlugin struct {
	sync.Mutex
	VendorName      string
	ApplicationName string

	userConfigFolder string
	codec            plugin.StandardMessageCodec
	databases        map[int32]*sql.DB // store database handlers
	databasePaths    map[int32]string  // store database file path
	databaseId       int32             // store max database id

	queryAsMapList bool
	debug          bool // debug mode
}

var _ flutter.Plugin = &SqflitePlugin{} // compile-time type check

// NewSqflitePlugin initialize the plugin
func NewSqflitePlugin(vendor, appName string) *SqflitePlugin {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	return &SqflitePlugin{
		VendorName:      vendor,
		ApplicationName: appName,
		databases:       make(map[int32]*sql.DB),
		databasePaths:   make(map[int32]string),
	}
}

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
	if p.debug {
		log.Println("home dir=", p.userConfigFolder)
	}

	channel := plugin.NewMethodChannel(messenger, channelName, plugin.StandardMethodCodec{})
	channel.HandleFunc(METHOD_INSERT, p.handleInsert)
	channel.HandleFunc(METHOD_BATCH, p.handleBatch)
	channel.HandleFunc(METHOD_DEBUG_MODE, p.handleDebugMode)
	channel.HandleFunc(METHOD_OPTIONS, p.handleOptions)
	channel.HandleFunc(METHOD_CLOSE_DATABASE, p.handleCloseDatabase)
	channel.HandleFunc(METHOD_OPEN_DATABASE, p.handleOpenDatabase)
	channel.HandleFunc(METHOD_EXECUTE, p.handleExecute)
	channel.HandleFunc(METHOD_UPDATE, p.handleUpdate)
	channel.HandleFunc(METHOD_QUERY, p.handleQuery)
	channel.HandleFunc(METHOD_GET_PLATFORM_VERSION, p.handleGetPlatformVersion)
	channel.HandleFunc(METHOD_GET_DATABASES_PATH, p.handleGetDatabasePath)
	channel.HandleFunc("deleteDatabase", p.handleDeleteDatabase)
	channel.HandleFunc("databaseExists", p.handleDatabaseExists)
	return nil
}

func (p *SqflitePlugin) handleGetPlatformVersion(arguments interface{}) (reply interface{}, err error) {
	version := fmt.Sprintf("%s %s", runtime.GOOS, runtime.GOARCH)
	return version, nil
}

func (p *SqflitePlugin) handleGetDatabasePath(arguments interface{}) (reply interface{}, err error) {
	return p.userConfigFolder, nil
}

// Not implemented
func (p *SqflitePlugin) handleOptions(arguments interface{}) (reply interface{}, err error) {
	var args map[string]interface{}
	var ok bool
	if args, ok = arguments.(map[string]interface{}); !ok {
		return nil, errors.New("invalid param for option call")
	}
	paramAsList, ok := args["PARAM_QUERY_AS_MAP_LIST"]
	if ok {
		p.queryAsMapList = paramAsList.(bool)
	}
	return nil, nil
}

func (p *SqflitePlugin) handleCloseDatabase(arguments interface{}) (reply interface{}, err error) {
	databaseId, db, err := p.getDatabase(arguments)
	if err != nil {
		return nil, err
	}
	err = db.Close()
	p.Lock()
	defer p.Unlock()
	delete(p.databasePaths, databaseId)
	delete(p.databases, databaseId)
	return nil, err
}

func (p *SqflitePlugin) handleOpenDatabase(arguments interface{}) (reply interface{}, err error) {
	// map[interface {}]interface {}{"path":"/Users/kael/Library/Application Support/libCachedImageData.db", "singleInstance":true}
	var ok bool
	var args map[interface{}]interface{}
	if args, ok = arguments.(map[interface{}]interface{}); !ok {
		return nil, errors.New("invalid arguments")
	}
	var dbpath string
	var readOnly bool
	var singleInstance bool
	if dpath, ok := args[PARAM_PATH]; ok {
		dbpath = dpath.(string)
	}
	if rdo, ok := args[PARAM_READ_ONLY]; ok {
		readOnly = rdo.(bool)
	}
	if si, ok := args[PARAM_SINGLE_INSTANCE]; ok {
		singleInstance = si.(bool) && MEMORY_DATABASE_PATH != dbpath
	}
	if dbpath == "" {
		log.Printf(errorFormat, "invalid dbpath")
		return nil, errors.New("invalid dbpath")
	}
	log.Println("dbpath=", dbpath)
	if readOnly {
		log.Printf(errorFormat, "readonly not supported")
	}
	if MEMORY_DATABASE_PATH != dbpath {
		err = os.MkdirAll(path.Dir(dbpath), 0755)
		if err != nil {
			log.Printf(errorFormat, err.Error())
		}
	}
	if singleInstance {
		dbId, ok := p.getDatabaseByPath(dbpath)
		if ok {
			return map[interface{}]interface{}{
				PARAM_ID:        dbId,
				PARAM_RECOVERED: true,
			}, nil
		}
	}
	var engine *sql.DB
	engine, err = sql.Open("sqlite3", dbpath)
	if err != nil {
		return makeError(err)
	}
	_, err = engine.Exec("VACUUM")
	if err != nil {
		return makeError(err)
	}

	p.Lock()
	defer p.Unlock()
	p.databaseId++
	p.databases[p.databaseId] = engine
	p.databasePaths[p.databaseId] = dbpath
	return map[interface{}]interface{}{
		PARAM_ID:        p.databaseId,
		PARAM_RECOVERED: false,
	}, nil
}

func (p *SqflitePlugin) handleInsert(arguments interface{}) (reply interface{}, err error) {
	_, db, err := p.getDatabase(arguments)
	if err != nil {
		return makeError(err)
	}
	sqlStr, args, err := p.getSqlCommand(arguments)
	if p.debug {
		log.Println("sql=", sqlStr, "args=", args)
	}
	if err != nil {
		return makeError(err)
	}
	result, err := db.Exec(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	return result.LastInsertId()
}

func (p *SqflitePlugin) handleBatch(arguments interface{}) (reply interface{}, err error) {
	_, db, err := p.getDatabase(arguments)
	if err != nil {
		return makeError(err)
	}
	args, ok := arguments.(map[interface{}]interface{})
	if !ok {
		return makeError(errors.New("invalid args"))
	}
	ioperations, ok := args[PARAM_OPERATIONS]
	if !ok {
		return makeError(errors.New("invalid operation"))
	}
	operations, ok := ioperations.([]interface{})
	if !ok {
		return makeError(errors.New("invalid operation data format"))
	}
	var noResult = false
	val, ok := args[PARAM_NO_RESULT]
	if ok {
		noResult = val.(bool)
	}
	var continueOnError = false
	val, ok = args[PARAM_CONTINUE_OR_ERROR]
	if ok {
		continueOnError = val.(bool)
	}

	var results []interface{}
	for _, operate := range operations {
		mtd, ok := operate.(map[interface{}]interface{})[PARAM_METHOD]
		if !ok {
			return makeError(errors.New("empty method"))
		}
		method, ok := mtd.(string)
		if !ok {
			return makeError(errors.New("invalid method"))
		}
		sqlStr, xargs, err := p.getSqlCommand(operate)
		if err != nil {
			return makeError(err)
		}
		switch method {

		case METHOD_INSERT:
			result, err := db.Exec(sqlStr, xargs...)
			if err != nil {
				errResult, err := makeError(err)
				if !continueOnError {
					return errResult, err
				}else{
					results = append(results, errResult)
					continue
				}
			}
			if !noResult {
				id, _ := result.LastInsertId()
				results = append(results, p.createBatchOperationResult(id))
			}

		case METHOD_UPDATE:
			result, err := db.Exec(sqlStr, xargs...)
			if err != nil {
				errResult, err := makeError(err)
				if !continueOnError {
					return errResult, err
				}else{
					results = append(results, errResult)
					continue
				}
			}
			if !noResult {
				rowsAffected, _ := result.RowsAffected()
				results = append(results, p.createBatchOperationResult(rowsAffected))
			}
		case METHOD_EXECUTE:
			_, err = db.Exec(sqlStr, xargs...)
			if err != nil {
				errResult, err := makeError(err)
				if !continueOnError {
					return errResult, err
				}else{
					results = append(results, errResult)
					continue
				}
			}
			if !noResult {
				results = append(results, p.createBatchOperationResult(nil))
			}
		case METHOD_QUERY:
			rows, err := db.Query(sqlStr, xargs...)
			if err != nil {
				errResult, err := makeError(err)
				if !continueOnError {
					return errResult, err
				}else{
					results = append(results, errResult)
					continue
				}
			}
			rowsReply, err := p.getRowsReply(rows)
			if !noResult {
				results = append(results, p.createBatchOperationResult(rowsReply))
			}

		default:
			return makeError(errors.New("Invalid batch param"))
		}
	}
	if noResult {
		return nil, nil
	} else {
		return results, nil
	}
}

func (p *SqflitePlugin) createBatchOperationResult(result interface{}) (map[interface{}]interface{}) {
	out := map[interface{}]interface{}{}
	out[PARAM_RESULT] = result
	return out
}

func makeError(err error) (map[interface{}]interface{}, error) {
	result := map[interface{}]interface{}{}

	errDetail := map[interface{}]interface{}{}
	errDetail["code"] = "sqlite_error"
	errDetail["message"] = err.Error()
	errDetail["data"] = "data"

	result["error"] = errDetail
	fmt.Println("error map", result)
	return result, err
}


func (p *SqflitePlugin) handleDebugMode(arguments interface{}) (reply interface{}, err error) {
	var v bool
	var ok bool
	if v, ok = arguments.(bool); !ok {
		return makeError(errors.New("Invalid argument type"))
	}

	p.debug = v

	// do nothing now
	return nil, nil
}

func (p *SqflitePlugin) handleExecute(arguments interface{}) (reply interface{}, err error) {
	_, db, err := p.getDatabase(arguments)
	if err != nil {
		return makeError(err)
	}
	sqlStr, args, err := p.getSqlCommand(arguments)
	if p.debug {
		log.Println("sql=", sqlStr, "args=", args)
	}
	if err != nil {
		return makeError(err)
	}
	var r sql.Result
	r, err = db.Exec(sqlStr, args...)
	if p.debug {
		log.Printf("result=%#v err=%v\n", r, err)
	}
	if err != nil {
		return makeError(err)
	}

	return nil, nil
}

func (p *SqflitePlugin) handleUpdate(arguments interface{}) (reply interface{}, err error) {
	_, db, err := p.getDatabase(arguments)
	if err != nil {
		return 0, err
	}
	sqlStr, args, err := p.getSqlCommand(arguments)
	if p.debug {
		log.Println("sql=", sqlStr, "args=", args)
	}
	if err != nil {
		return makeError(err)
	}
	result, err := db.Exec(sqlStr, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (p *SqflitePlugin) handleQuery(arguments interface{}) (reply interface{}, err error) {
	_, db, err := p.getDatabase(arguments)
	if err != nil {
		return makeError(err)
	}
	sqlStr, args, err := p.getSqlCommand(arguments)
	if p.debug {
		log.Println("sql=", sqlStr, "args=", args)
	}
	if err != nil {
		return makeError(err)
	}
	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return makeError(err)
	}
	return p.getRowsReply(rows)
}

func (p *SqflitePlugin) handleDatabaseExists(arguments interface{}) (reply interface{}, err error) {
	return false, nil
}

func (p *SqflitePlugin) handleDeleteDatabase(arguments interface{}) (reply interface{}, err error) {
	if dbPath, ok := arguments.(string); ok {
		if dbPath != MEMORY_DATABASE_PATH {
			err = os.Remove(dbPath)
		}
	}
	return makeError(err)
}

func (p *SqflitePlugin) getDatabase(arguments interface{}) (int32, *sql.DB, error) {
	var args map[interface{}]interface{}
	var ok bool
	if args, ok = arguments.(map[interface{}]interface{}); !ok {
		return -1, nil, errors.New("db not found")
	}
	if dbId, ok := args[PARAM_ID]; ok {
		p.Lock()
		defer p.Unlock()
		id, ok := dbId.(int32)
		if !ok {
			return -1, nil, errors.New("Invaid db id")
		}
		if db, ok := p.databases[id]; ok {
			return id, db, nil
		}
	}
	return -1, nil, errors.New("invalid database")
}

func (p *SqflitePlugin) getDatabaseByPath(dbPath string) (int32, bool) {
	if dbPath == MEMORY_DATABASE_PATH {
		return -1, false
	}
	p.Lock()
	defer p.Unlock()
	for id, pt := range p.databasePaths {
		if pt == dbPath {
			return id, true
		}
	}
	return -1, false
}

func (p *SqflitePlugin) getSqlCommand(arguments interface{}) (sqlStr string, xargs []interface{}, err error) {
	var args map[interface{}]interface{}
	var ok bool
	if args, ok = arguments.(map[interface{}]interface{}); !ok {
		return "", nil, errors.New("db not found")
	}
	tsql, ok := args[PARAM_SQL]
	if !ok {
		return "", nil, errors.New("SQL is not set")
	}
	sqlStr = tsql.(string)
	if sqlStr == "" {
		return "", nil, errors.New("SQL is empty")
	}
	targs, ok := args[PARAM_SQL_ARGUMENTS]
	if ok && targs != nil {
		xargs, _ = targs.([]interface{})
	}
	return
}

func (p *SqflitePlugin) getRowsReply(rows *sql.Rows) (rowsReply interface{}, err error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var resultRows []interface{}
	for {
		if !rows.Next() {
			break
		}
		var resultRow []interface{}
		dest := make([]interface{}, len(cols))
		for k, _ := range cols {
			var ignore interface{}
			dest[k] = &ignore
		}
		err = rows.Scan(dest...)
		for _, cval := range dest {
			var val interface{}
			val = *cval.(*interface{})
			var out interface{}

			if (val == nil){
				out = nil
			}else {
				switch val.(type) {
					case []byte:
						out = string(val.([]byte))
					default:
						out = val
				}
			}
			resultRow = append(resultRow, out)
		}
		//log.Printf("resultrow=%#v\n", resultRow)
		resultRows = append(resultRows, resultRow)
	}
	var icols []interface{}
	for _, col := range cols {
		icols = append(icols, col)
	}
	return map[interface{}]interface{}{
		"columns": icols,
		"rows":    resultRows,
	}, nil
}
