package utils_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/stretchr/testify/require"
)

type withDate struct {
	Date utils.Date `json:"date"`
}

func TestDate(t *testing.T) {

	d1, err := utils.StrToDate("2022-07-28")
	if err != nil {
		t.Fatalf("failed to parse date: %s", err)
	}

	expected := 20220728
	if int(d1) != expected {
		t.Fatalf("Invalid date: expected %d, got %d", expected, int(d1))
	}

	if d1.Year() != 2022 {
		t.Fatalf("Invalid month: expected %d, got %d", 2022, d1.Year())
	}
	if d1.Month() != 7 {
		t.Fatalf("Invalid month: expected %d, got %d", 7, d1.Month())
	}
	if d1.Day() != 28 {
		t.Fatalf("Invalid month: expected %d, got %d", 28, d1.Day())
	}

	if d1.String() != "2022-07-28" {
		t.Fatalf("Invalid formatting: expected %s, got %s", "2022-07-28", d1.String())
	}

	if d1.StringRu() != "28.07.2022" {
		t.Fatalf("Invalid formatting: expected %s, got %s", "28.07.2022", d1.StringRu())
	}
	if d1.StringRuShort() != "28.07.22" {
		t.Fatalf("Invalid formatting: expected %s, got %s", "28.07.22", d1.StringRuShort())
	}
	if d1.AsNumber() != "20220728" {
		t.Fatalf("Invalid formatting: expected %s, got %s", "20220728", d1.AsNumber())
	}

	var d2 utils.Date
	if !d2.IsNil() {
		t.Fatalf("Expected nil date")
	}
	if d1.IsNil() {
		t.Fatalf("Expected not nil date")
	}

	t1 := d1.Time()
	d2.SetTime(t1)
	if d1 != d2 {
		t.Fatalf("Invalid setting from time: expected %s, got %s", d1.String(), d2.String())
	}

	wd := &withDate{}
	jsonStr1 := `{"date":"2022-07-28"}`
	err = json.Unmarshal([]byte(jsonStr1), wd)
	if err != nil {
		t.Fatalf("failed to unmarshal json: %s", err)
	}

	if wd.Date != d1 {
		t.Fatalf("Invalid json date: expected %s, got %s", d1.String(), wd.Date.String())
	}

	jsonBytes, err := json.Marshal(wd)
	if err != nil {
		t.Fatalf("failed to marshal json: %s", err)
	}
	jsonStr2 := string(jsonBytes)
	if jsonStr2 != jsonStr1 {
		t.Fatalf("Invalid json string: expected %s, got %s", jsonStr1, jsonStr2)
	}

	t3 := time.Date(2022, 07, 28, 15, 37, 54, 0, time.UTC)
	d3 := utils.DateOfTime(t3)
	if d3 != d1 {
		t.Fatalf("Invalid date of time: expected %s, got %s", d1.String(), d3.String())
	}

	d1, err = utils.StrToDate("2023-05-22")
	if err != nil {
		t.Fatalf("failed to parse date: %s", err)
	}
	expected = 20230522
	if int(d1) != expected {
		t.Fatalf("Invalid date: expected %d, got %d", expected, int(d1))
	}
}

func TestTimezone(t *testing.T) {

	IANA_TIME_ZONE := "Europe/Moscow"
	_, err := time.LoadLocation(IANA_TIME_ZONE)
	require.NoError(t, err)

}
