package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"gobook/arguments"
	"gobook/inpx"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	_ "github.com/mattn/go-sqlite3"
)

func SQLiteInitDB() {
	output, err := inpx.Open(arguments.Config.Inpx)
	if err != nil {
		log.Fatalf("error when open inpx: %v", err)
	}
	exist := true
	if _, err := os.Stat(arguments.Config.DB_CONN); os.IsNotExist(err) {
		exist = false
		log.Printf("SQLite not exist, create the file on path: %v", arguments.Config.DB_CONN)
		os.Create(arguments.Config.DB_CONN)
	}
	db, err := sql.Open("sqlite3", arguments.Config.DB_CONN)
	defer db.Close()
	if err != nil {
		log.Fatal("Can't open DB: ", err)
	}
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		db.Close()
		log.Fatalf("Can't set up parametr REFERENCES in DB: %v", err)
	}
	// check if quantity books in db equal quantity books in inpx
	var needInit bool
	if exist {
		needInit = checkdb(output, db)
	} else {
		needInit = true
		SQLiteReInitDB()
	}
	if needInit {
		_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS books (
    LibId       INTEGER PRIMARY KEY,
    Authors     TEXT,   -- массив строк
    Genres      TEXT,   -- массив строк
    Title       TEXT,
    Series      TEXT,
    SeriesNum   INTEGER, -- массив целых чисел
    FileName    TEXT,
    FileExt     TEXT,
    FileDir     TEXT,
    FileArchive TEXT,
    FileSize    INTEGER,
    Deleted     BOOLEAN,
    Date        TIMESTAMP,
    Lang        TEXT
    )`)
		if err != nil {
			db.Close()
			log.Fatalf("Can't create table books:\n%s", err)
		} else {
			log.Printf("Successfully create table: %s", "Books")
		}
		_, err = db.Exec(`
    CREATE table IF NOT EXISTS Users (
        id BIGINT PRIMARY KEY,
        FirstName TEXT,
        LastName TEXT,
        Username TEXT
    )`)
		if err != nil {
			db.Close()
			log.Fatalf("Can't create table Users:\n%s", err)
		} else {
			log.Printf("Successfully create table: %s", "Users")
		}
		_, err = db.Exec(`
    CREATE table IF NOT EXISTS request(
        id serial PRIMARY KEY,
        Chat INTEGER REFERENCES users(id),
        Message INTEGER,
        Callback TEXT,
        WebIdent TEXT
    )`)
		if err != nil {
			db.Close()
			log.Fatalf("Can't create table request:\n%s", err)
		} else {
			log.Printf("Successfully create table: %s", "Request")
		}
		_, err = db.Exec(`
    CREATE table IF NOT EXISTS ui (
        id serial PRIMARY KEY, 
        Chat INTEGER REFERENCES users(id),
        WebIdent TEXT,
        Dir TEXT,
        FileInArchive TEXT,
        Title TEXT,
        SeriesNums TEXT,
        Authors TEXT,
        Ganres TEXT,
        Series TEXT
    )`)
		if err != nil {
			db.Close()
			log.Fatalf("Can't create table ui:\n%s", err)
		} else {
			log.Printf("Successfully create table: %s", "Ui")
		}
		// for Progress of work
		count := 1
		var all int
		for _, value := range output.Archives {
			all += len(value)
		}

		fmt.Println("Started database initialization")
		start := time.Now()
		for _, data := range output.Archives {
			for i := range data {
				//transform all arrays into json to store in DB
				var AuthorsJSON []byte
				var GanresJSON []byte
				var SeriesNumJSON []byte
				for _, v := range data[i].Authors {
					AuthorsJSON, err = json.Marshal(v.Name)
				}
				if err != nil {
					log.Printf("Can't Marshal authors: %v", err)
				}
				if len(data[i].Genres) > 0 {
					GanresJSON, err = json.Marshal(data[i].Genres)
					if err != nil {
						log.Printf("Can't Marshal ganres: %v", err)
					}
				} else {
					GanresJSON = []byte(`{}`)
				}
				if len(data[i].SeriesNum) > 0 {
					SeriesNumJSON, err = json.Marshal(data[i].SeriesNum)
					if err != nil {
						log.Printf("Can't Marshal seriesnums: %v", err)
					}
				} else {
					SeriesNumJSON = []byte(`{}`)
				}
				_, err = db.Exec(`INSERT INTO books (LibId, Authors, Genres, Title, Series, SeriesNum, FileName, FileExt, FileDir, FileArchive, FileSize, Deleted, Date, Lang) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) ON CONFLICT (LibId) DO NOTHING`,
					data[i].LibId,
					string(AuthorsJSON),
					string(GanresJSON),
					data[i].Title,
					data[i].Series,
					string(SeriesNumJSON),
					data[i].File.Name,
					data[i].File.Ext,
					data[i].File.Dir,
					data[i].File.Archive,
					data[i].File.Size,
					data[i].Deleted,
					data[i].Date,
					data[i].Lang,
				)
				if err != nil {
					log.Printf("Error when add data: %v", err)
				}
				if count%(all/100) == 0 {
					fmt.Printf("Progress: %d%%\n", (100*count)/all)
				}
				count++
			}
		}
		elapsed := time.Since(start)
		fmt.Printf("\nDatabase is Inited\nIt took: %v\n", elapsed)
	}
}

func (p *Pagenator) SQLiteFind(field string, request []string, lang string) []Extract {
	db, err := sql.Open("sqlite3", arguments.Config.DB_CONN)
	var req string
	if len(request) > 1 {
		req = strings.Join(request, " ")
	} else {
		req = request[0]
	}
	if err != nil {
		log.Printf("Can't connect to DB: %v", err)
	}
	defer db.Close()
	var response *sql.Rows
	caser := cases.Title(language.English)
	switch field {
	case "Author":
		var reqDB string
		if len(request) == 1 {
			reqDB = `select FileDir, FileArchive, FileName, Title, SeriesNum, Authors, Genres, Series from books, json_each(books.Authors) where json_each.value like '%` + caser.String(request[0]) + `%'`
		} else {
			reqDB = `SELECT FileDir, FileArchive, FileName, Title, SeriesNum, Authors, Genres, Series FROM books WHERE EXISTS (SELECT 1 FROM json_each(books.Authors) WHERE json_each.value like '%` + caser.String(request[0]) + `%')`
			for _, v := range request[1:] {
				reqDB = reqDB + ` AND EXISTS (SELECT 1 FROM json_each(books.Authors) WHERE json_each.value LIKE '%` + caser.String(v) + `%')`
			}
		}
		response, err = db.Query(reqDB + "and lang = '" + lang + "'")
	case "Series":
		response, err = db.Query(`select FileDir, FileArchive, FileName, Title, SeriesNum, Authors, Genres, Series from books where LOWER(series) like '%` + caser.String(req) + `%'` + "and lang = '" + lang + "'")
	case "Title":
		response, err = db.Query(`select FileDir, FileArchive, FileName, Title, SeriesNum, Authors, Genres, Series from books where LOWER(title) like '%` + caser.String(req) + `%'` + "and lang = '" + lang + "'")
		response, err = db.Query(`select FileDir, FileArchive, FileName, Title, SeriesNum, Authors, Genres, Series from books where LOWER(title) like '%` + caser.String(req) + `%'` + `and lang = '` + lang + `'`)
	}
	if err != nil {
		log.Printf("Can't work with request: %v", err)
	}
	var finded []Extract
	type b struct {
		FileDir       string
		FileInArchive string
		FileName      string
		Title         string
		SeriesNum     []int
		Authors       []string
		Genres        []string
		Series        string
	}
	for response.Next() {
		var b b
		var seriesNumStr, authorsStr, genresStr string
		err = response.Scan(&b.FileDir, &b.FileInArchive, &b.FileName, &b.Title, &seriesNumStr, &authorsStr, &genresStr, &b.Series)
		if seriesNumStr != "{}" {
			err = json.Unmarshal([]byte(seriesNumStr), &b.SeriesNum)
			if err != nil {
				log.Printf("Can't umarshal response from seriesNumStr from DB: %v", err)
			}
		}

		if authorsStr != "" {
			err = json.Unmarshal([]byte(authorsStr), &b.Authors)
			if err != nil {
				log.Printf("Can't Unmarshal response Author from DB: %v", err)
			}
		}

		if genresStr != "" {
			err = json.Unmarshal([]byte(genresStr), &b.Genres)
			if err != nil {
				log.Printf("Can't Unmarshal response Genres from DB: %v", err)
			}
		}
		var seriesnums string
		for _, i := range b.SeriesNum {
			seriesnums += fmt.Sprint(i)
		}
		find := Extract{
			Dir:           b.FileDir + "/" + b.FileInArchive + ".zip",
			FileInArchive: b.FileName + ".fb2",
			Title:         b.Title,
			SeriesNums:    caser.String(seriesnums),
			Authors:       caser.String(strings.Join(b.Authors, " ")),
			Ganres:        caser.String(strings.Join(b.Genres, ",")),
			Series:        b.Series,
		}
		finded = append(finded, find)
	}
	p.SQLiteWriteRequest(finded)
	return finded
}

// Write request from user to UI table
func (p *Pagenator) SQLiteWriteUserInfo() {
	db, err := sql.Open("sqlite3", arguments.Config.DB_CONN)
	if err != nil {
		log.Printf("Can't connect to DB: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`INSERT INTO Users (id, FirstName, LastName, Username) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO NOTHING`, p.Chat, p.FirstName, p.LastName, p.Username)

	if err != nil {
		log.Printf("Can't wrtie users data on DB: %v", err)
	}
	_, err = db.Exec(`INSERT INTO request (Chat, Message, Callback) VALUES ($1, $2, $3)`, p.Chat, p.Message, p.Callback)

	if err != nil {
		log.Printf("Can't wrtie request data on DB: %v", err)
	}
}

// Wipe request's from user in UI table
func (p *Pagenator) SQLiteWipeUserInfo() {
	db, err := sql.Open("sqlite3", arguments.Config.DB_CONN)
	if err != nil {
		log.Printf("Can't connect to DB: %v", err)
	}
	_, err = db.Exec(`DELETE FROM request WHERE Chat = $1`, p.Chat)
	if err != nil {
		log.Printf("Can't delete request's from DB: %v", err)
	}
}

// Read finded books and send it to user
func (p *Pagenator) SQLiteReadUserInfo() []Pagenator {
	db, err := sql.Open("sqlite3", arguments.Config.DB_CONN)
	if err != nil {
		log.Printf("Can't connect to DB: %v", err)
	}
	response, err := db.Query(`SELECT Chat, Message, Callback from request where Chat = $1`, p.Chat)

	var data []Pagenator
	var b Pagenator
	count := 1
	for response.Next() {
		err = response.Scan(&b.Chat, &b.Message, &b.Callback)
		data = append(data, b)
		count++
		if err != nil {
			log.Printf("Can't Scan data for paginator ui: %v", err)
		}
	}
	return data
}

// Write request's from finded books
func (p *Pagenator) SQLiteWriteRequest(extract []Extract) {
	db, err := sql.Open("sqlite3", arguments.Config.DB_CONN)
	if err != nil {
		log.Printf("Can't connect to DB: %v", err)
	}
	defer db.Close()
	for i := range extract {
		if p.Chat != 0 {
			_, err = db.Exec(`INSERT INTO Users (id, FirstName, LastName, Username) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO NOTHING`, p.Chat, p.FirstName, p.LastName, p.Username)
			if err != nil {
				log.Printf("Can't wrtie users data on DB: %v", err)
			}
		}
		_, err = db.Exec(`INSERT INTO ui (Chat, Dir, FileInArchive, Title, SeriesNums, Authors, Ganres, Series) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			p.Chat,
			extract[i].Dir,
			extract[i].FileInArchive,
			extract[i].Title,
			extract[i].SeriesNums,
			extract[i].Authors,
			extract[i].Ganres,
			extract[i].Series,
		)
		if err != nil {
			log.Printf("Can't wrtie users data on DB: %v", err)
		} else if p.WebIdent != "" {
			_, err = db.Exec(`INSERT INTO ui (WebIdent, Dir, FileInArchive, Title, SeriesNums, Authors, Ganres, Series) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
				p.WebIdent,
				extract[i].Dir,
				extract[i].FileInArchive,
				extract[i].Title,
				extract[i].SeriesNums,
				extract[i].Authors,
				extract[i].Ganres,
				extract[i].Series,
			)
			if err != nil {
				log.Printf("Can't wrtie users data in ui: %v", err)
			}
		}
	}
}
func (p *Pagenator) SQLiteReadRequest() []Extract {
	db, err := sql.Open("sqlite3", arguments.Config.DB_CONN)
	if err != nil {
		log.Printf("Can't connect to DB: %v", err)
	}
	defer db.Close()
	var response *sql.Rows
	if p.Chat != 0 {
		response, err = db.Query(`select Dir, FileInArchive, Title, SeriesNums, Authors, Ganres, Series from ui where Chat =  $1`, p.Chat)
		if err != nil {
			log.Printf("Can't work with request: %v", err)
		}
	}
	if p.WebIdent != "" {
		response, err = db.Query(`select Dir, FileInArchive, Title, SeriesNums, Authors, Ganres, Series from ui where WebIdent =  $1`, p.WebIdent)
		if err != nil {
			log.Printf("Can't work with request: %v", err)
		}
	}
	var finded []Extract
	for response.Next() {
		var b Extract
		err = response.Scan(&b.Dir, &b.FileInArchive, &b.Title, &b.SeriesNums, &b.Authors, &b.Ganres, &b.Series)
		if err != nil {
			log.Printf("Error when parse the response: %v", err)
		}
		var seriesnums string
		for _, i := range b.SeriesNums {
			seriesnums += fmt.Sprintf("%v ", i)
		}
		finded = append(finded, b)
	}
	return finded
}

func SQLiteReInitDB() {
	db, err := sql.Open("sqlite3", arguments.Config.DB_CONN)
	if err != nil {
		log.Printf("Can't connect to DB: %v", err)
	}
	defer db.Close()
	var count int
    //check if db already exist
	err = db.QueryRow(`SELECT count(*) 
        FROM sqlite_master 
        WHERE type='table' 
        AND name='books'`).Scan(&count)
	if err != nil {
		panic(err)
	}
	if count > 0 {
		_, err = db.Query(`delete from books`)
		if err != nil {
			log.Printf("Can't delete data from books: %v", err)
		}
	}
}

// Wipe all ui request's from user
func (p *Pagenator) SQLiteWipeRequest() {
	db, err := sql.Open("sqlite3", arguments.Config.DB_CONN)
	if err != nil {
		log.Printf("Can't connect to DB: %v", err)
	}
	defer db.Close()
	if p.Chat != 0 {
		_, err = db.Exec(`DELETE FROM ui WHERE Chat = $1`, p.Chat)
		if err != nil {
			log.Printf("Can't delete from ui: %v", err)
		}
	} else if p.WebIdent != "" {
		_, err = db.Exec(`DELETE FROM ui WHERE WebIdent = $1`, p.WebIdent)
		if err != nil {
			log.Printf("Can't delete from ui: %v", err)
		}
	}
}

func SQLiteRequestCheck(ID int64) bool {
	db, err := sql.Open("sqlite3", arguments.Config.DB_CONN)
	if err != nil {
		log.Printf("Can't connect to DB: %v", err)
	}
	defer db.Close()
	var count int
	err = db.QueryRow(`select count(*) from ui where Chat = $1`, ID).Scan(&count)
	if count > 0 {
		return false
	} else {
		return true
	}
}
func checkdb(output *inpx.Index, db *sql.DB) bool {
	// for quantity books in inpx and db
	var all int
	var count int
	// for checking quantity of matches
	var check int
	// check if db file exist
	for _, value := range output.Archives {
		all += len(value)
	}
	err := db.QueryRow(`select count(*) from books`).Scan(&count)
	if err != nil {
		log.Printf("Can't count books in db: %v", err)
	}
	// check if quantity of books equal quantity in inpx
	log.Println("Start check if books in inpx equal in db")
	log.Println("Start test with random values")
	for i := 0; i < 10; i++ {
		random := rand.Intn(all)
		// variable to check if title equal. Not empty for case when initial var may be equal if nothing is found.
		book := "this is book variable"
		inpx := "this is inpx variable"
		err := db.QueryRow(`select Title from books where LibId = $1`, random).Scan(&book)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("When check db found empty id. This is not a problem, but needs to be log if in future you will have any problem with founded books. Id: %d", random)
			} else {
				log.Printf("When test quantity was err: %v", err)
			}
		}
	found:
		for _, data := range output.Archives {
			for i := range data {
				if random == data[i].LibId {
					inpx = data[i].Title
					break found
				}
			}
		}
		if book == inpx {
			check++
		}
	}
	if check >= 3 {
		log.Println("Random check is pass.\nStart check with last values.")
		check = 0
	} else {
		log.Println("Check is not Pass. Start initialization")
		return true
	}
	lenofArchives := 0
	for _, data := range output.Archives {
		length := len(data)
		inpx := data[length-1].Title
		inpxID := data[length-1].LibId
		var book string
		err := db.QueryRow(`select Title from books where LibId = $1`, inpxID).Scan(&book)
		if err != nil {
			log.Printf("When check db was error : %v", err)
		}
		if book == inpx {
			check++
		}
		lenofArchives++
	}
	if check == lenofArchives {
		log.Println("db already set and have correct values. Don't need to reinit. Start bot")
		return false
	}
	log.Println("Check is not Pass. Start initialization")
	return true
}
