package db

import (
	"database/sql"
	"fmt"
	"gobook/arguments"
	"gobook/inpx"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Extract struct {
	Dir           string `json:"dir"`
	FileInArchive string `json:"fileinarchive"`
	Title         string `json:"title"`
	SeriesNums    string `json:"seriesnms"`
	Authors       string `json:"authors"`
	Ganres        string `json:"ganres"`
	Series        string `json:"series"`
}

type Pagenator struct {
	Chat      int64
	Message   int
	Callback  string
	FirstName string
	LastName  string
	Username  string
	WebIdent  string
}

func PostgresInitDB() {
	db, err := sql.Open("postgres", arguments.Config.DB_CONN)
	pgcheck := arguments.Config.DB_URL + ":" + arguments.Config.DB_PORT
	timeout := 1 * time.Minute
	retryInterval := 5 * time.Second

	startTime := time.Now()

	for time.Since(startTime) < timeout {
		conn, err := net.DialTimeout("tcp", pgcheck, retryInterval)
		if err == nil {
			conn.Close()
			fmt.Println("PostgreSQL is available!")
			break
		}

		fmt.Println("PostgreSQL is not yet available. Retrying in 5 seconds...")
		time.Sleep(retryInterval)
	}
	defer db.Close()
	if err != nil {
		db.Close()
		log.Fatal("Can't open DB: ", err)
	}
	output, err := inpx.Open(arguments.Config.Inpx)
	if err != nil {
		db.Close()
		log.Fatalf("error when open inpx: %v", err)
	}
	needInit := checkdb(output, db)
	if needInit {
		PostgresReInitDB()
	}
	if needInit {
		_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS books (
    LibId       INTEGER PRIMARY KEY,
    Authors     TEXT[],   -- массив строк
    Genres      TEXT[],   -- массив строк
    Title       TEXT,
    Series      TEXT,
    SeriesNum   INTEGER[], -- массив целых чисел
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
        Chat BIGINT REFERENCES users(id),
        Message INTEGER,
        Callback TEXT,
        WebIdent TEXT
    )`)
		if err != nil {
			db.Close()
			log.Fatalf("Can't create table request:\n%s", err)
		} else {
			log.Printf("Successfully create table: %s", "request")
		}
		_, err = db.Exec(`
    CREATE table IF NOT EXISTS ui (
        id serial PRIMARY KEY, 
        Chat BIGINT REFERENCES users(id),
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
		var wg sync.WaitGroup
		var mutex sync.Mutex
		start := time.Now()
		jobs := make(chan []inpx.Book, 90)
		for i := 0; i < cap(jobs); i++ {
			go func() {
				for data := range jobs {
					for i := range data {
						_, err := db.Exec(`INSERT INTO books (LibId, Authors, Genres, Title, Series, SeriesNum, FileName, FileExt, FileDir, FileArchive, FileSize, Deleted, Date, Lang) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) ON CONFLICT (LibId) DO NOTHING`,
							data[i].LibId,
							pq.Array(authorNames(data[i].Authors)),
							pq.Array(data[i].Genres),
							data[i].Title,
							data[i].Series,
							pq.Array(data[i].SeriesNum),
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
						mutex.Lock()
						if count%(all/100) == 0 {
							fmt.Printf("Progress: %d%%\n", (100*count)/all)
						}
						count++
						mutex.Unlock()
					}
					wg.Done()
				}
			}()
		}
		for _, value := range output.Archives {
			jobs <- value
			wg.Add(1)
		}
		wg.Wait()
		close(jobs)
		elapsed := time.Since(start)
		fmt.Printf("\nDatabase is Inited\nIt took: %v\n", elapsed)
	}
}

func authorNames(names []inpx.Author) []string {
	var authors []string
	for _, author := range names {
		for _, name := range author.Name {
			authors = append(authors, strings.ToLower(name))
		}
	}
	return authors
}

func (p *Pagenator) PostgresFind(field string, request []string, lang string) []Extract {
	db, err := sql.Open("postgres", arguments.Config.DB_CONN)
	var req string
	caser := cases.Title(language.English)
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
	switch field {
	case "Author":
		var reqDB string
		if len(request) == 1 {
			reqDB = `select FileDir, FileArchive, FileName, Title, SeriesNum, Authors, Genres, Series from books where authors::text like '%` + strings.ToLower(request[0]) + `%'`
		} else {
			reqDB = `select FileDir, FileArchive, FileName, Title, SeriesNum, Authors, Genres, Series from books where authors::text like '%` + strings.ToLower(request[0]) + `%'`
			for _, v := range request[1:] {
				reqDB = reqDB + ` AND authors::text like '%` + strings.ToLower(v) + `%'`
			}
		}
		response, err = db.Query(reqDB + "and lang = '" + lang + "'")
	case "Series":
		response, err = db.Query(`select FileDir, FileArchive, FileName, Title, SeriesNum, Authors, Genres, Series from books where LOWER(series) like '%` + strings.ToLower(req) + `%'` + "and lang = '" + lang + "'")
	case "Title":
		response, err = db.Query(`select FileDir, FileArchive, FileName, Title, SeriesNum, Authors, Genres, Series from books where LOWER(title) like '%` + strings.ToLower(req) + `%'` + "and lang = '" + lang + "'")
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
		SeriesNum     []string
		Authors       []string
		Genres        []string
		Series        string
	}

	for response.Next() {
		var b b
		err = response.Scan(&b.FileDir, &b.FileInArchive, &b.FileName, &b.Title, pq.Array(&b.SeriesNum), pq.Array(&b.Authors), pq.Array(&b.Genres), &b.Series)
		if err != nil {
			log.Printf("Error when parse the response: %v\n value: %v", err, b.SeriesNum)
		}
		var seriesnums string
		for _, i := range b.SeriesNum {
			seriesnums += fmt.Sprintf("%v ", i)
		}
		find := Extract{
			Dir:           b.FileDir + "/" + b.FileInArchive + ".zip",
			FileInArchive: b.FileName + ".fb2",
			Title:         b.Title,
			SeriesNums:    caser.String(seriesnums),
			Authors:       caser.String(strings.Join(b.Authors, ", ")),
			Ganres:        caser.String(strings.Join(b.Genres, ", ")),
			Series:        b.Series,
		}
		p.PostgresWriteRequest(find)
		finded = append(finded, find)
	}
	return finded
}

// Write request from user to UI table
func (p *Pagenator) PostgresWriteUserInfo() {
	db, err := sql.Open("postgres", arguments.Config.DB_CONN)
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
func (p *Pagenator) PostgresWipeUserInfo() {
	db, err := sql.Open("postgres", arguments.Config.DB_CONN)
	if err != nil {
		log.Printf("Can't connect to DB: %v", err)
	}
	_, err = db.Exec(`DELETE FROM request WHERE Chat = $1`, p.Chat)
	if err != nil {
		log.Printf("Can't delete request's from DB: %v", err)
	}
}

// Read finded books and send it to user
func (p *Pagenator) PostgresReadUserInfo() []Pagenator {
	db, err := sql.Open("postgres", arguments.Config.DB_CONN)
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
func (p *Pagenator) PostgresWriteRequest(extract Extract) {
	db, err := sql.Open("postgres", arguments.Config.DB_CONN)
	if err != nil {
		log.Printf("Can't connect to DB: %v", err)
	}
	defer db.Close()
	if p.Chat != 0 {
		_, err = db.Exec(`INSERT INTO Users (id, FirstName, LastName, Username) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO NOTHING`, p.Chat, p.FirstName, p.LastName, p.Username)
		if err != nil {
			log.Printf("Can't wrtie users data on DB: %v", err)
		}
		_, err = db.Exec(`INSERT INTO ui (Chat, Dir, FileInArchive, Title, SeriesNums, Authors, Ganres, Series) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			p.Chat,
			extract.Dir,
			extract.FileInArchive,
			extract.Title,
			extract.SeriesNums,
			extract.Authors,
			extract.Ganres,
			extract.Series,
		)
		if err != nil {
			log.Printf("Can't wrtie users data on DB: %v", err)
		}
	} else if p.WebIdent != "" {
		_, err = db.Exec(`INSERT INTO ui (WebIdent, Dir, FileInArchive, Title, SeriesNums, Authors, Ganres, Series) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			p.WebIdent,
			extract.Dir,
			extract.FileInArchive,
			extract.Title,
			extract.SeriesNums,
			extract.Authors,
			extract.Ganres,
			extract.Series,
		)
		if err != nil {
			log.Printf("Can't wrtie users data in ui: %v", err)
		}
	}
}
func (p *Pagenator) PostgresReadRequest() []Extract {
	db, err := sql.Open("postgres", arguments.Config.DB_CONN)
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

func PostgresReInitDB() {
	db, err := sql.Open("postgres", arguments.Config.DB_CONN)
	if err != nil {
		db.Close()
		log.Fatalf("Can't connect to DB: %v", err)
	}
	defer db.Close()
	var exists bool
	err = db.QueryRow(`SELECT EXISTS (
        SELECT 1 
        FROM   pg_tables
        WHERE  schemaname = 'public' 
        AND    tablename = 'books'
    )`).Scan(&exists)
	if err != nil {
		panic(err)
	}
	if exists {
		_, err = db.Query(`delete from books`)
		if err != nil {
			db.Close()
			log.Fatalf("Can't delete data from books: %v", err)
		}
	}
}

// Wipe all ui request's from user
func (p *Pagenator) PostgresWipeRequest() {
	db, err := sql.Open("postgres", arguments.Config.DB_CONN)
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

func PostgresRequestCheck(id int64) bool {
	db, err := sql.Open("postgres", arguments.Config.DB_CONN)
	if err != nil {
		log.Printf("Can't connect to DB: %v", err)
	}
	defer db.Close()
	var count int
	err = db.QueryRow(`select count(*) from ui where Chat = $1`, id).Scan(&count)
	if count > 0 {
		return false
	}
	return true
}
