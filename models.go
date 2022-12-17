package main

const randomFoxAPIURL = "https://randomfox.ca/floof"

func purrbotAPIURL(nsfw bool, imageType string, animated bool) string {
	url := "https://purrbot.site/api"
	if nsfw {
		url += "/nsfw"
	}
	url += "/" + imageType
	if animated {
		url += "/gif"
	}
	return url
}

type purrbotAPIResponse struct {
	Error bool   `json:"error"`
	Link  string `json:"link"`
	Time  int    `json:"time"`
}

type randomfoxAPIResponse struct {
	Image string `json:"image"`
	Link  string `json:"link"`
}
