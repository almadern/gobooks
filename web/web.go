package web

import (
	"archive/zip"
	"fmt"
	"gobook/arguments"
	db "gobook/database"
	"gobook/zipextract"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// start web page
func StartWeb() {
	http.HandleFunc("/", HomePage)
	http.HandleFunc("/found", FoundFunc)
	http.HandleFunc("/download", DownloadFunc)

	log.Printf("Start server on port: %s", arguments.Config.Port)
	http.ListenAndServe(arguments.Config.Port, nil)
}

func HomePage(res http.ResponseWriter, req *http.Request) {
	id := uuid.New()
	send := struct {
		WebIdent string
	}{WebIdent: fmt.Sprintf("%v", id)}
	err := arguments.Config.Mainpage.Execute(res, send)
	if err != nil {
		log.Printf("Error when paste the data to / page: %v", err)
	}
}

func FoundFunc(res http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	searchType := req.FormValue("search_type")
	searchQuery := req.FormValue("search_query")
	format := req.FormValue("format")
	language := req.FormValue("language")
	WebIdent := req.FormValue("webIdent")

	log.Println("User send request search_type: ", searchType, "search_query: ", searchQuery, "format: ", format, "language: ", language, "webIdent: ", WebIdent)
	user := db.Pagenator{
		WebIdent: WebIdent,
	}
	var output []db.Extract
	switch arguments.Config.DB {
	case "file":
		output = user.FileFind(arguments.Config.Inpx, searchType, strings.Split(searchQuery, " "), language)
	case "postgres":
		output = user.PostgresFind(searchType, strings.Split(searchQuery, " "), language)
	case "sqlite":
		output = user.SQLiteFind(searchType, strings.Split(searchQuery, " "), language)
	}

	if len(output) == 0 {
		arguments.Config.EmptyRes.Execute(res, nil)
	} else {
		switch arguments.Config.DB {
		case "file":
			user.FileWrtieUserInfo()
		case "postgres":
			user.PostgresWriteUserInfo()
		}
	}
	send := struct {
		WebIdent string
		Format   string
		Output   []db.Extract
	}{WebIdent: WebIdent, Format: format, Output: output}

	err := arguments.Config.Findpage.Execute(res, send)
	if err != nil {
		log.Printf("Error when paste the data to /found page: %v", err)
	}
}

func DownloadFunc(res http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	request := req.Form["choice"]
	format := req.FormValue("format")
	webIdent := req.FormValue("webIdent")
	fmt.Printf("request: %v", request)
	user := db.Pagenator{
		WebIdent: webIdent,
	}
	var decodedData []db.Extract
	switch arguments.Config.DB {
	case "file":
		decodedData = user.FileReadRequest()
	case "postgres":
		decodedData = user.PostgresReadRequest()
	case "sqlite":
		decodedData = user.SQLiteReadRequest()
	}
	outFile, err := os.Create(webIdent + ".zip")
	if err != nil {
		log.Printf("Can't create zip archive: %v", err)
	}
	defer outFile.Close()
	zipWriter := zip.NewWriter(outFile)

	for _, choose := range request {
		value, err := strconv.Atoi(choose)
		output := decodedData[value]
		title := ""
		name := output.Title
		if len(output.SeriesNums) > 2 {
			title = output.SeriesNums + "-" + name
		} else {
			title = name
		}
		nametofb2 := fmt.Sprintf("%v", title) + ".fb2"
		nameepub := fmt.Sprintf("%v", title) + ".epub"
		if len(output.SeriesNums) > 2 {
			title = output.SeriesNums + "-" + name
		} else {
			title = name
		}

		err = zipextract.Open(output.Dir, output.FileInArchive, nametofb2)
		if err == nil {
			if format == "epub" {
				var cmd *exec.Cmd
				switch os := runtime.GOOS; os {
				case "darwin":
					cmd = exec.Command(arguments.Config.CONVERTER_PATH+"/fb2c_mac", "convert", nametofb2)
				case "windows":
					cmd = exec.Command(arguments.Config.CONVERTER_PATH+"\fb2c.exe", "convert", nametofb2)
				case "linux":
					cmd = exec.Command(arguments.Config.CONVERTER_PATH+"/fb2c_linux", "convert", nametofb2)
				}
				err = cmd.Run()
				if err != nil {
					log.Printf("When exec converter error: %v", err)
				}
				zipextract.AddFileToZIP(zipWriter, nameepub)
				err = os.Remove(nameepub)
				if err != nil {
					log.Printf("Error when delete file %v", err)
				}
				err := os.Remove(nametofb2)
				if err != nil {
					log.Printf("Error when delete file %v", err)
				}
			} else if format == "fb2" {
				zipextract.AddFileToZIP(zipWriter, nametofb2)
				err := os.Remove(nametofb2)
				if err != nil {
					log.Printf("Error when delete file %v", err)
				}
			}
		}
	}
	err = zipWriter.Close()
	if err != nil {
		log.Printf("Error when close zip archive: %v", err)
	}
	_, err = outFile.Seek(0, 0)
	if err != nil {
		log.Printf("Error when change position in zip archive: %v", err)
	}
	// Send header to user
	res.Header().Set("Content-Type", "application/zip")
	res.Header().Set("Content-Disposition", `attachment; filename="books.zip"`)
	io.Copy(res, outFile)
	// Delete files after send
	err = os.Remove(webIdent + ".zip")
	err = os.Remove("conversion.log")
	if err != nil {
		log.Printf("Error when delete logs from converter")
	}

	if err != nil {
		log.Printf("Error when remove ziparchive: %v", err)
	}
	switch arguments.Config.DB {
	case "file":
		user.FileWipeRequestandUserInfo()
	case "postgres":
		user.PostgresWipeUserInfo()
		user.PostgresWipeRequest()
	case "sqlite":
		user.SQLiteWipeUserInfo()
		user.SQLiteWipeRequest()
	}
}
