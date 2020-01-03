package actions

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

type reuploadAction struct {
	options *Options
}

func newReuploadAction() *reuploadAction {
	return &reuploadAction{
		options: &Options{
			Name: "reupload",
			Re: regexp.MustCompile(
				`(https?:\/\/)?((damn-community\.com|screenshots\.relentless\.wtf|puddelgaming\.de\/upload)\/.*\.(png|jpe?g))`,
			),
		},
	}
}

func (a reuploadAction) GetOptions() *Options {
	return a.options
}

func (a reuploadAction) Run(e *Event) error {
	link := e.Match[2]

	link = "https://" + link

	newLink, err := reupload(link, e.ImgurClientID, "ChronophylosBot/"+e.CHB3Version)
	if err != nil {
		e.Log.Error().
			Err(err).
			Str("link", link).
			Msg("Reuploading an image to imgur")
	} else {
		e.Log.Info().
			Str("link", link).
			Msg("Reuploading an image to imgur")
		e.Say("Did you mean " + newLink + " ?")
	}

	return nil
}

type imgurBody struct {
	Data struct {
		Link  string `json:"link"`
		Error string `json:"error"`
	} `json:"data"`
	Success bool `json:"success"`
}

func reupload(link string, clientID, userAgent string) (string, error) {
	client := &http.Client{}

	form := url.Values{}
	form.Add("image", link)

	req, err := http.NewRequest(
		"POST",
		"https://api.imgur.com/3/upload",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf(
			"could not make POST request for https://api.imgur.com/3/upload: %v",
			err,
		)
	}

	req.Header.Add("Authorization", "Client-ID "+clientID)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not post to imgur: %v", err)
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not bytes from response: %v", err)
	}

	log.Debug().Bytes("data", bytes).Send()

	var body imgurBody
	if err = json.Unmarshal(bytes, &body); err != nil {
		return "", err
	}

	if !body.Success {
		return "", fmt.Errorf("imgur api returned: %s", body.Data.Error)
	}

	return body.Data.Link, nil
}
