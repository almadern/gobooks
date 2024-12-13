package db

import (
	"bufio"
	"encoding/json"
	"fmt"
	"gobook/arguments"
	"gobook/inpx"
	"log"
	"os"
	"strings"
)

// change the message to propriet display in telegram
func (p *Pagenator) FileFind(path string, field string, request []string, lang string) []Extract {
	output, err := inpx.Open(path)
	if err != nil {
		log.Fatalf("error when open inpx: %v", err)
	}
	finded, _ := inpx.FindByField(output, field, request, lang)
	var filepaths []Extract
	i := 1
	for _, value := range *finded {
		authorNames := make([]string, len(value.Authors))
		for i, author := range value.Authors {
			authorNames[i] = strings.Join(author.Name, " ")
		}
		var seriesnum string
		for _, i := range value.SeriesNum {
			seriesnum += fmt.Sprintln(i)
		}
		authors := strings.Join(authorNames, ", ")
		ganres := strings.Join(value.Genres, ", ")
		title := value.Title
		series := value.Series
		filepaths = append(filepaths, Extract{
			Dir:           value.File.Dir + "/" + value.File.Archive + ".zip",
			FileInArchive: value.File.Name + ".fb2",
			Title:         title,
			SeriesNums:    seriesnum,
			Authors:       authors,
			Ganres:        ganres,
			Series:        series,
		})
		i++
	}
		// create(or open file) by user ID to store searched books
		// userID for normal filename in OS(in full used $ and \n)
		options := os.O_WRONLY | os.O_APPEND | os.O_CREATE
		var userID string
		if p.Chat != 0 {
			userID = fmt.Sprintf("%v", p.Chat)
		} else if p.WebIdent != "" {
			userID = fmt.Sprintf("%v", p.WebIdent)
		}

		file, err := os.OpenFile(userID, options, os.FileMode(0600))
		defer file.Close()
		encoder := json.NewEncoder(file)
		for _, value := range filepaths {
			err = encoder.Encode(value)
			if err != nil {
				log.Printf("Error when write to file: %v", err)
				return nil
			}
		}
	p.FileWrtieUserInfo()
	return filepaths
}

// create file to store paginator info to close it when send books or delete request
func (p *Pagenator) FileWrtieUserInfo() {
	var userID string
	if p.Chat != 0 {
		userID = fmt.Sprintf("%v", p.Chat)
	} else if p.WebIdent != "" {
		userID = fmt.Sprintf("%v", p.WebIdent)
	}
	options := os.O_WRONLY | os.O_APPEND | os.O_CREATE
	ui, err := os.OpenFile(userID+"_ui", options, os.FileMode(0600))
	if err != nil {
		log.Printf("Error when open to file ui: %v", err)
	}
	uiencoder := json.NewEncoder(ui)
	err = uiencoder.Encode(p)
	if err != nil {
		log.Printf("Error when write to file ui: %v", err)
	}
	defer ui.Close()
}

func (p *Pagenator) FileWipeUserInfo() error {
	var userID string
	if p.Chat != 0 {
		userID = fmt.Sprintf("%v", p.Chat)
	} else if p.WebIdent != "" {
		userID = fmt.Sprintf("%v", p.WebIdent)
	}
	file, err := os.Open(userID + "_ui")
	if err != nil {
		log.Printf("Error when read file: %v", err)
	}
	defer file.Close()
	err = os.Truncate(userID+"_ui", 0)
	if err != nil {
		log.Println("Error when empty file: ", err)
		return err
	}
	return nil
}

func (p *Pagenator) FileWipeRequestandUserInfo() {
	var userID string
	if p.Chat != 0 {
		userID = fmt.Sprintf("%v", p.Chat)
		err := os.Remove(userID)
		if err != nil {
			log.Println("Error when empty file: ", err)
			return
		}
		err = os.Remove(userID + "_ui")
		if err != nil {
			log.Println("Error when empty file: ", err)
			return
		}
	} else if p.WebIdent != "" {
		userID = fmt.Sprintf("%v", p.WebIdent)
		err := os.Remove(userID)
		if err != nil {
			log.Println("Error when empty file: ", err)
			return
		}
		err = os.Remove(userID + "_ui")
		if err != nil {
			log.Println("Error when empty file: ", err)
			return
		}
	}
}

func FileRequestCheck(id int64) bool {
	userID := fmt.Sprintf("%v", id)
	filename := userID
	size, err := os.Stat(filename)
	if err != nil && !os.IsNotExist(err) {
		log.Printf("Error when getting file size %v", err)
	} else if os.IsNotExist(err) {
		os.Create(filename)
	}
	if size.Size() > 0 {
		return false
	}
	return true
}

func (p *Pagenator) FileReadUserInfo() []Pagenator {
	var userID string
	if p.Chat != 0 {
		userID = fmt.Sprintf("%v", p.Chat)
	} else if p.WebIdent != "" {
		userID = fmt.Sprintf("%v", p.WebIdent)
	}
	file, err := os.Open(userID + "_ui")
	if err != nil {
		log.Printf("Error when read file: %v", err)
	}
	defer file.Close()
	var decodedData []Pagenator
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		var Extract Pagenator
		if err := json.Unmarshal([]byte(line), &Extract); err != nil {
			log.Printf("Error when decode JSON: %v", err)
			return nil
		}
		decodedData = append(decodedData, Extract)
	}
	return decodedData
}

func (p *Pagenator) FileReadRequest() []Extract {
	var userID string
	if p.Chat != 0 {
		userID = fmt.Sprintf("%v", p.Chat)
	} else if p.WebIdent != "" {
		userID = fmt.Sprintf("%v", p.WebIdent)
	}
	file, err := os.Open(userID)
	if err != nil {
		log.Printf("Error when read file: %v", err)
	}
	defer file.Close()
	var decodedData []Extract
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		var Extract Extract
		if err := json.Unmarshal([]byte(line), &Extract); err != nil {
			log.Printf("Error when decode JSON: %v", err)
			return nil
		}
		decodedData = append(decodedData, Extract)
	}
	return decodedData
}
