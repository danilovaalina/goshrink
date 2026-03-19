package model

type Analytics struct {
	ShortCode   string
	TotalClicks int
	ByDay       []DailyStat
	ByUserAgent []UserAgentStat
}

type DailyStat struct {
	Date  string
	Count int
}

type UserAgentStat struct {
	UserAgent string
	Count     int
}
