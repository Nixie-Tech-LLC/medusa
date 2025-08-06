package model

type Prayer struct {
  Name   string // “FAJR”, “DHUHR”, …
  Time   string // “05:12”
  Period string // “AM” or “PM”
  Iqama  string // if you also compute iqama times
}

type AthanPageData struct {
  City    string
  Date    string       // “AUGUST 5, 2025”
  Prayers []Prayer
}

