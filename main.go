package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"os"
	"os/exec"
	"strings"
	"time"
)

const jsonFilename = "data/covidNasjonalt.json"
const csvFilename = "data/NasjonalCovidData.csv"
const smittetallCsvFilename = "data/CovidSmitteTall.csv"

func main() {
	createDataFolder()

	haveDownloaded := false

	var datastruct covidNasjonalt
	data, err := readFromJsonFile(jsonFilename) // reads the stored data from earlier query
	if err != nil {                             // should download if error
		fmt.Println("got:", err, "\nWe download instead of reading from stored data file")
		// panic("To stop from downloading")
		data = getDataFromApi() // will panic if it fails
		haveDownloaded = true

		writeBinaryDataToFile(data) // creates json file
	} else {
		fmt.Println("read Data from stored json file")
	}

	err = json.Unmarshal(data, &datastruct)
	if err != nil {
		panic(err)
	}

	if !dataUpToDate() { // always return false
		smittetall := getSmittetall()
		writeSmitteTallData(smittetall)
	}

	lastRecordedDate, _ := time.Parse("2006-01-02T15:04:05", datastruct.Registreringer[len(datastruct.Registreringer)-1].Dato)

	if isSameDate(lastRecordedDate, time.Now()) {
		fmt.Println("Data is now up to date")
		writeStructToCsv(datastruct)
		createPlotFileWithPyplot()
	} else {
		fmt.Println("Current data does not have current day!")
		if haveDownloaded {
			writeStructToCsv(datastruct)
			createPlotFileWithPyplot()
			log.Fatal("We have downloaded the latest data, but it is not up to date. This probably means helsedir has not published today yet (latest they should publish is 1300)")
		} else {
			data = getDataFromApi()
			haveDownloaded = true
			err = json.Unmarshal(data, &datastruct)
			if err != nil {
				panic(err)
			}
			writeStructToCsv(datastruct)
			lastRecordedDate, _ = time.Parse("2006-01-02T15:04:05", datastruct.Registreringer[len(datastruct.Registreringer)-1].Dato)
			fmt.Println("Last date of the data we have is :", datastruct.Registreringer[len(datastruct.Registreringer)-1].Dato, "And today is:", time.Now())
			if !isSameDate(lastRecordedDate, time.Now()) {
				fmt.Println("data is probably not published yet. (between 0000 and 1300)")
			} else {
				writeBinaryDataToFile(data)
				fmt.Println("data is now updated!")
			}
			createPlotFileWithPyplot()
			time.Sleep(time.Second * 3)
		}

	}
}

// dataUpToDate is not implemented
func dataUpToDate() bool {
	return false
}

func createPlotFileWithPyplot() {
	workingDirectory, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	command := exec.Command("python.exe", workingDirectory+`/visualizeCovidData.py`)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err = command.Run()
	if err != nil {
		panic(err)
	}
}

type covidNasjonalt struct {
	Registreringer []struct {
		AntInnlagte   int64  `json:"antInnlagte"`
		AntRespirator int64  `json:"antRespirator"`
		Dato          string `json:"dato"`
	} `json:"registreringer"`
}

type smitteTallPerDag struct {
	Tekst  string `json:"tekst"`
	Antall int64  `json:"antall"`
	Dato   string `json:"fordeltPaa"`
}

func getSmittetall() []smitteTallPerDag {
	response, err := http.Get("https://statistikk.fhi.no/api/msis/etterDiagnoseFordeltPaaProvedato?fraOpprettetdato=2020-02-21&tilOpprettetdato=2021-04-10&diagnoseKodeListe=713")
	if err != nil {
		panic(err)
	}
	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}
	var smitteTallsData []smitteTallPerDag
	err = json.Unmarshal(responseData, &smitteTallsData)
	if err != nil {
		panic(err)
	}
	return smitteTallsData
}

func writeSmitteTallData(smittetall []smitteTallPerDag) {

	stringToWrite := "Antall registrerte smittede med corona per dato,dato\n"

	for _, dataDAG := range smittetall {
		stringToWrite += fmt.Sprintln(fmt.Sprint(dataDAG.Antall) + "," + dataDAG.Dato)
	}
	err := os.WriteFile(smittetallCsvFilename, []byte(stringToWrite), 0666)
	if err != nil {
		log.Fatal(err)
	}
}

func isSameDate(date1, date2 time.Time) bool {
	year1, month1, day1 := date1.Date()
	year2, month2, day2 := date2.Date()
	return year1 == year2 && month1 == month2 && day1 == day2
}

func getDataFromApi() []byte {
	fmt.Println("Trying to download")
	configFile, err := os.Open("config.txt")
	if err != nil {
		panic(err)
	}

	configScanner := bufio.NewScanner(configFile)
	configScanner.Scan()
	configData := strings.Split(configScanner.Text(), ":")
	if len(configData) != 2 {
		panic("file config.txt (should be located in same folder as exe file) has wrong format, is missing or api key is missing.\n should be 'apiKey:apiKeyHereWithNoSpacesOrQuotes'.\nGet api key at https://utvikler.helsedirektoratet.no/products/covid19")
	}
	apiKey := configData[1]

	Client := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}

	req, err := http.NewRequest(http.MethodGet, "https://api.helsedirektoratet.no/ProduktCovid19/Covid19statistikk/nasjonalt", nil)
	req.Header.Set("Ocp-Apim-Subscription-Key", apiKey)
	if err != nil {
		panic(err)
	}

	res, err := Client.Do(req)

	if res.StatusCode != 200 {
		panic(res.Status + "\nSomething Went wrong when trying to download the data.\nDouble check your api key in the config.txt, then blame the developer for creating a horrible script!")
	}
	if err != nil {
		panic(err)
	}
	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}
	return body
}

func writeBinaryDataToFile(data []byte) {
	file, err := os.Create(jsonFilename)
	if err != nil {
		panic(err)
	}
	file.Write(data)
}

func readFromJsonFile(filename string) ([]byte, error) {
	file, err := os.Open(filename) // json data lastet ned fra apien.
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func writeStructToCsv(data covidNasjonalt) {
	csvFile, err := os.OpenFile(csvFilename, os.O_RDWR|os.O_CREATE, os.FileMode(0666))
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	stringToWrite := "dato,innlagte,respirator\n"

	if err != nil {
		panic(err)
	}
	for _, dataDAG := range data.Registreringer {
		stringToWrite += fmt.Sprintln(dataDAG.Dato + "," + fmt.Sprint(dataDAG.AntInnlagte) + "," + fmt.Sprint(dataDAG.AntRespirator))
	}
	// fmt.Println(stringToWrite)
	err = os.WriteFile(csvFilename, []byte(stringToWrite), 0666)
	os.Mkdir("data", 0666)
	if err != nil {
		log.Fatal(err)
	}
}

func createDataFolder() {
	err := os.Mkdir("data", 0666)
	if !errors.Is(err, os.ErrExist) && err != nil {
		panic(err)
	}
}
