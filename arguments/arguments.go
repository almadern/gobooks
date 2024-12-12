package arguments

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v2"
)

// globar struct for using as config from user
type ConfigStruct struct {
	DB             string  `yaml:"DB" toml:"DB"`
	DB_DB          string  `yaml:"DB_DB" toml:"DB_DB"`
	DB_USER        string  `yaml:"DB_USER" toml:"DB_USER"`
	DB_URL         string  `yaml:"DB_URL" toml:"DB_URL"`
	DB_PORT        string  `yaml:"DB_PORT" toml:"DB_PORT"`
	DB_PASSWORD    string  `yaml:"DB_PASSWORD" toml:"DB_PASSWORD"`
	Inpx           string  `yaml:"Inpx" toml:"Inpx"`
	Web            bool    `yaml:"Web" toml:"Web"`
	Port           string  `yaml:"Port" toml:"Port"`
	Token          string  `yaml:"Token" toml:"Token"`
	Telegram       bool    `yaml:"Telegram" toml:"Telegram"`
	DB_CONN        string  `yaml:"DB_CONN" toml:"DB_CONN"` // don't will be use in command line arg's. This is will be used as variable in code
	CONVERTER_PATH string  `yaml:"CONVERTER_PATH" toml:"CONVERTER_PATH"`
	Restore        bool    `yaml:"RESTORE" toml:"RESTORE"`     // variable for function to restore previus request
	BLACKLIST      string  `yaml:"BLACKLIST" toml:"BLACKLIST"` // vaiable for blocked user to access bot
	WHITELIST      string  `yaml:"WHITELIST" toml:"WHITELIST"` // virable to
	AllAccess      bool    // all can acess bot, check only blacklist
	StrictAcc      bool    // only user in Whitelist can access. In other case bot will not responde
	BlackList      []int64 // array of user id who will be blocked
	Whitelist      []int64 // array of user id who can access
	LANGUAGE       string  `yaml:"LANGUAGE" toml:"LANGUAGE"` // default language for search and bot response messages
    HTML_PATH      string  `yaml:"HTML_PATH" toml:"HTML_PATH"`
	Mainpage       *template.Template
	Findpage       *template.Template
	EmptyRes       *template.Template
}

var Config ConfigStruct
var Configpath string
// set up initial parameters for programm
func Initialization() {
	flag.StringVar(&Config.DB, "DB", "postgres", "choose the DB to store data. Accetpt: postgres, Inpx")
	flag.StringVar(&Config.DB, "db", "postgres", "choose the DB to store data. Accetpt: postgres, Inpx")
	flag.StringVar(&Config.DB, "database", "postgres", "choose the DB to store data. Accetpt: postgres, Inpx")
	flag.StringVar(&Config.DB_DB, "DB_DB", "", "Database when data will store")
	flag.StringVar(&Config.DB_USER, "DB_USER", "postgres", "Database user(default: postgres)")
	flag.StringVar(&Config.DB_URL, "DB_URL", "localhost", "Database url to connect(default: localhost)")
	flag.StringVar(&Config.DB_PORT, "DB_PORT", "5432", "Database port to connect(default: 5432)")
	flag.StringVar(&Config.DB_PASSWORD, "DB_PASSWORD", "", "Database password to connect(by default empty)")
	flag.StringVar(&Config.Inpx, "inpx", "", "set the path to Inpx file")
	flag.BoolVar(&Config.Web, "web", false, "set this param to set Web page to find books separit from bot. Set -Web to set it to true")
	flag.BoolVar(&Config.Web, "w", false, "set this param to set Web page to find books separit from bot. Set -Web to set it to true")
	flag.BoolVar(&Config.Telegram, "telegram", true, "set this param to start Telegram bot")
	flag.BoolVar(&Config.Telegram, "T", true, "set this param to start Telegram bot")
	flag.StringVar(&Config.Token, "token", "", "Token needs to start bot")
	flag.StringVar(&Config.Token, "t", "", "Token needs to start bot")
	flag.StringVar(&Config.Port, "port", "", "Port is command to set Port to start Web server. default Port - 80")
	flag.StringVar(&Config.Port, "p", "", "Port is command to set Port to start Web server. default Port - 80")
	flag.StringVar(&Configpath, "config", "", "Path to config file(accept TOML, YAML)")
	flag.StringVar(&Config.CONVERTER_PATH, "CONVERTER_PATH", "", "CONVERTER_PATH in need to be set if converter store in different directory(programm look for dir named converter)")
    flag.StringVar(&Config.HTML_PATH, "html", "", "path to html folder from repository. Needs to be set to propper programm work")
	flag.BoolVar(&Config.Restore, "restore", false, "Option for restore previus request, if it don't been finish")
	flag.StringVar(&Config.LANGUAGE, "language", "en", "Option for set language in 2 letter format(en, ru etc.)- bot will responde in this lang and default lang for search(default is en, bot can responde on ru and en)")

	flag.Parse()
	SetVariable()
}

func SetVariable() {
	if Configpath != "" {
		format := filepath.Ext(Configpath)
		data, err := os.ReadFile(Configpath)
		if err != nil {
			log.Fatalf("Error when open config file: %v", err)
		} else {
			log.Printf("Open config file: %v", Configpath)
		}
		switch format {
		case ".toml":
			err = toml.Unmarshal(data, &Config)
		case ".yaml", ".yml":
			err = yaml.Unmarshal(data, &Config)
		case ".json":
			err = json.Unmarshal(data, &Config)
		default:
			fmt.Println("File format is not supported\nSupported: TOML, YAML")
			os.Exit(1)
		}
		if err != nil {
			log.Fatalf("Error when decode config file: %v", err)
		}
	} else if os.Getenv("DB") != "" {
		Config.DB = os.Getenv("DB")
		if os.Getenv("POSTGRES_DB") != "" {
			Config.DB_DB = os.Getenv("POSTGRES_DB")
		} else {
			Config.DB_DB = os.Getenv("DB_DB")
		}
		if os.Getenv("POSTGRES_USER") != "" {
			Config.DB_USER = os.Getenv("POSTGRES_USER")
		} else {
			Config.DB_USER = os.Getenv("DB_USER")
		}
		if os.Getenv("POSTGRES_PASSWORD") != "" {
			Config.DB_PASSWORD = os.Getenv("POSTGRES_PASSWORD")
		} else {
			Config.DB_PASSWORD = os.Getenv("DB_PASSWORD")
		}
		Config.DB_PORT = os.Getenv("DB_PORT")
		Config.Inpx = os.Getenv("INPX")
		Config.Web = getEnvAsBool("WEB", false)
		Config.Telegram = getEnvAsBool("TELEGRAM", true)
		Config.Port = os.Getenv("WEB_PORT")
		Config.Token = os.Getenv("TOKEN")
		Config.CONVERTER_PATH = os.Getenv("CONVERTER_PATH")
		Config.DB_URL = os.Getenv("DB_URL")
		Config.Restore = getEnvAsBool("RESTORE", false)
		Config.BLACKLIST = os.Getenv("BLACKLIST")
		Config.WHITELIST = os.Getenv("WHITELIST")
		Config.LANGUAGE = os.Getenv("LANGUAGE")
        Config.HTML_PATH = os.Getenv("HTML_PATH")
	}
	switch Config.DB {
	case "file":
		fmt.Println("Set inpx database ", Config.DB)
	case "postgres":
		if os.Getenv("DATABASE_URL") != "" {
			Config.DB_CONN = os.Getenv("DATABASE_URL")
		} else if Config.DB_PASSWORD == "" {
			Config.DB_CONN = fmt.Sprintf("postgres://%s@%s/%s?sslmode=disable", Config.DB_USER, Config.DB_URL+":"+Config.DB_PORT, Config.DB_DB)
		} else {
			Config.DB_CONN = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", Config.DB_USER, Config.DB_PASSWORD, Config.DB_URL+":"+Config.DB_PORT, Config.DB_DB)
		}
	case "sqlite":
		fmt.Println("Set sqlite database: ", Config.DB_DB)
		Config.DB_CONN = Config.DB_DB
	default:
		fmt.Println("Value is not accepteble. Wrtie -help to see supported database")
		os.Exit(1)
	}
	if Config.Inpx == "" {
		fmt.Println("Path needs to be set")
		os.Exit(1)
    } else {
		if _, err := os.Stat(Config.Inpx); err != nil {
			fmt.Println("Can't open file", err)
			os.Exit(1)
		}
	}
	switch Config.Token {
	case "":
		if Config.Telegram {
			fmt.Println("Token needs to be set to start Telegram")
			os.Exit(1)
		}
	}
	switch Config.Port {
	case "":
		if Config.Web {
			fmt.Println("you don't set Port for Web")
			os.Exit(1)
		}
	default:
		if string(Config.Port[0]) != ":" {
			Config.Port = ":" + Config.Port
		}
	}
	// set access to bot. 3 mode allowed - all access with blacklist, strict access only by Whitelist, no strict
	if Config.BLACKLIST != "" {
		Config.AllAccess = true
		Config.StrictAcc = false
		str := strings.Split(Config.BLACKLIST, ",")
		Config.BlackList = make([]int64, len(str))
		for i := range Config.BlackList {
			Config.BlackList[i], _ = strconv.ParseInt(str[i], 10, 64)
		}
	} else if Config.WHITELIST != "" {
		Config.AllAccess = false
		Config.StrictAcc = true
		str := strings.Split(Config.WHITELIST, ",")
		Config.Whitelist = make([]int64, len(str))
		for i := range Config.Whitelist {
			Config.Whitelist[i], _ = strconv.ParseInt(str[i], 10, 64)
		}
	} else {
		// set value to false to give access without any Strict
		Config.AllAccess, Config.StrictAcc = true, true
	}

	if Config.LANGUAGE == "" || len(Config.LANGUAGE) != 2 {
		Config.LANGUAGE = "en"
		log.Printf("Langueage don't set or its not in 2 letters format. By default will be set: en")
	}

	switch Config.CONVERTER_PATH {
	case "":
		pathinpx := filepath.Dir(Config.Inpx)
		dirPath := filepath.Join(pathinpx, "converter")
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			log.Printf("Can't set path to converter by Inpx file. Maybe converter in another dir. Please set the path manually by set CONVERTER_PATH in env or in config parth: %v", err)
			os.Exit(1)
		}
		Config.CONVERTER_PATH = dirPath
	default:
		dirName := filepath.Base(Config.CONVERTER_PATH)
		if dirName != "converter" {
			fmt.Println("Can't find the converter dir in set path")
			os.Exit(1)
		}
	}
	if check, err := os.Stat(Config.HTML_PATH); !check.IsDir() {
		log.Printf("Can't set path to html folder. Maybe html in another dir. Please set correct path in HTML_PATH in env: %v", err)
		os.Exit(1)
	}
	switch Config.LANGUAGE {
	case "ru":
		Config.Mainpage = template.Must(template.ParseFiles(Config.HTML_PATH + "/" + "index_ru.html"))
		Config.Findpage = template.Must(template.ParseFiles(Config.HTML_PATH + "/" + "found_ru.html"))
		Config.EmptyRes = template.Must(template.ParseFiles(Config.HTML_PATH + "/" + "empty_ru.html"))
	default:
		Config.Mainpage = template.Must(template.ParseFiles(Config.HTML_PATH + "/" + "index.html"))
		Config.Findpage = template.Must(template.ParseFiles(Config.HTML_PATH + "/" + "found.html"))
		Config.EmptyRes = template.Must(template.ParseFiles(Config.HTML_PATH + "/" + "empty.html"))
	}

	var access string
	if Config.AllAccess && Config.StrictAcc {
		access = "Access, without restriction"
	} else if Config.AllAccess {
		access = "All can access, who not in blacklist"
	} else if Config.StrictAcc {
		access = "Access granted only for users in whitelist"
	}

	log.Printf("Config is valid, start the work.\nConfiguration params:\n Database type: %s\n Web: %v\t Port: %v\n Telegram: %v \nDefault language: %s \n Restore previus request: %v\nAccess Mode: %s", Config.DB, Config.Web, Config.Port, Config.Telegram, Config.LANGUAGE, Config.Restore, access)
}

func getEnvAsBool(name string, defaultVal bool) bool {
	valStr := os.Getenv(name)
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	return defaultVal
}
