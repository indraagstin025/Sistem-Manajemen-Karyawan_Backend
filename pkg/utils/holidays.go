

package util

import (
	"Sistem-Manajemen-Karyawan/models" // Pastikan import models.Holiday
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// HolidayAPIData adalah struct helper untuk parsing JSON dari API
type HolidayAPIData struct {
	Date              string `json:"holiday_date"`
	Name              string `json:"holiday_name"`
	IsNationalHoliday bool   `json:"is_national_holiday"`
}

// GetHolidayMap mengambil data hari libur dari API eksternal dalam bentuk map.
func GetHolidayMap(year string) (map[string]bool, error) {
	holidayMap := make(map[string]bool)
	resp, err := http.Get("https://api-harilibur.vercel.app/api?year=" + year)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rawHolidays []HolidayAPIData
	if err := json.Unmarshal(body, &rawHolidays); err != nil {
		return nil, err
	}

	for _, rawHoliday := range rawHolidays {
		if rawHoliday.IsNationalHoliday {
			holidayMap[rawHoliday.Date] = true
		}
	}
	return holidayMap, nil
}

// GetExternalHolidaysForFrontend mengambil data hari libur dari API dalam bentuk slice.
func GetExternalHolidaysForFrontend(year string) ([]models.Holiday, error) {
	resp, err := http.Get("https://api-harilibur.vercel.app/api?year=" + year)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rawHolidays []HolidayAPIData
	if err := json.Unmarshal(body, &rawHolidays); err != nil {
		return nil, err
	}

	var holidays []models.Holiday
	for _, rawHoliday := range rawHolidays {
		if rawHoliday.IsNationalHoliday {
			holidays = append(holidays, models.Holiday{
				Date: rawHoliday.Date,
				Name: rawHoliday.Name,
			})
		}
	}
	return holidays, nil
}