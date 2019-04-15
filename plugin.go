package sqflite

import (
	"os"
	"path"
	"path/filepath"
	"runtime"

	flutter "github.com/go-flutter-desktop/go-flutter"
	"github.com/go-flutter-desktop/go-flutter/plugin"
	"github.com/go-xorm/xorm"
	"github.com/pkg/errors"
	_ "github.com/mattn/go-sqlite3"
	homedir "github.com/mitchellh/go-homedir"
)

const channelName = "com.tekartik.sqflite"

type SqflitePlugin struct {
	VendorName      string
	ApplicationName string

	userConfigFolder string
	codec            plugin.StandardMessageCodec
	engine *xorm.Engine
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
	case "darwin":home, err := homedir.Dir()
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
	channel.HandleFunc("deleteDatabase", p.handleDeleteDatabase)
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
	var ok bool
	var args map[string]interface{}
	if args,ok=arguments.(map[string]interface{});!ok {
		return nil, errors.New("invalid arguments")
	}
	var dbpath string
	if dpath,ok:=args["path"];ok {
		dbpath = dpath.(string)
	}
	if dbpath=="" {
		return nil, errors.New("invalid dbpath")
	}
	p.engine, err = xorm.NewEngine("sqlite3", path.Join(p.userConfigFolder, dbpath))
	return filepath.Join(p.userConfigFolder, p.VendorName, p.ApplicationName), nil
}

func (p *SqflitePlugin) handleOpenDatabase(arguments interface{}) (reply interface{}, err error) {
	return filepath.Join(p.userConfigFolder, p.VendorName, p.ApplicationName), nil
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

func (p *SqflitePlugin) handleDeleteDatabase(arguments interface{}) (reply interface{}, err error) {
	return filepath.Join(p.userConfigFolder, p.VendorName, p.ApplicationName), nil
}
