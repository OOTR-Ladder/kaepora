// Package ootrapi is an API client for the ootrandomizer.com HTTP API v2.
package ootrapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"time"

	"golang.org/x/time/rate"
)

// API holds the necessary state to communicate with the OOTR API.
type API struct {
	http    http.Client
	key     string
	limiter *rate.Limiter
}

// New creates a new authenticated, rate-limited access point to the API.
func New(key string) *API {
	return &API{
		// We're allowed 20 requests per 10 second
		limiter: rate.NewLimiter(20/10, 1),
		key:     key,
		http: http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SeedStatus represents the step at which seedgen currently is for a seed.
type SeedStatus int

// Possible values for SeedStatus.
const (
	// SeedStatusInvalid is our internal invalid default value, the API zero
	// value being a valid value.
	SeedStatusInvalid SeedStatus = -1

	SeedStatusGenerating   SeedStatus = 0 // in progress
	SeedStatusDone         SeedStatus = 1 // seed available to download
	SeedStatusDoneWithLink SeedStatus = 2 // "not possible from API"
	SeedStatusFailed       SeedStatus = 3 // generation failed, this won't fail gracefully for now
)

// IsValid returns false is the value can't be returned by the API.
func (s SeedStatus) IsValid() bool {
	return s >= 0 && s <= 3
}

func (api *API) getURL(subPath string, q url.Values) string {
	q.Set("key", api.key)

	u := url.URL{
		Scheme:   "https",
		Host:     "ootrandomizer.com",
		Path:     path.Join("/api/v2", subPath),
		RawQuery: q.Encode(),
	}
	return u.String()
}

// CreateSeed starts a new seedgen using the given OOTR version and settings.
func (api *API) CreateSeed(version string, settings map[string]interface{}) (string, error) {
	log.Printf("debug: creating API seed for version %s", version)

	body, err := json.Marshal(settings)
	if err != nil {
		return "", err
	}

	url := api.getURL("/seed/create", url.Values{
		"version": {version},
		"locked":  {"1"},
	})
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")

	var response struct {
		ID       string `json:"id"`
		Version  string `json:"version"`
		Spoilers bool   `json:"spoilers"`
	}
	if err := api.do(request, &response); err != nil {
		return "", err
	}

	if response.Spoilers {
		log.Printf("warning: API ignored the locked parameter")
	}
	if response.Version != version {
		log.Printf("warning: API version mismatch, expected '%s' got '%s'", version, response.Version)
	}

	log.Printf("debug: API got seed ID %s", response.ID)

	return response.ID, nil
}

// GetSeedStatus returns the step at which seedgen is for a given seed ID.
func (api *API) GetSeedStatus(id string) (SeedStatus, error) {
	log.Printf("debug: fetching API seed status for ID  %s", id)

	url := api.getURL("/seed/status", url.Values{"id": {id}})
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return SeedStatusInvalid, err
	}

	var res struct {
		Status   SeedStatus `json:"status"`
		Progress int        `json:"progress"` // 0-100
		// ignored: version, positionQueue, maxWaitTime, isMultiWorld
	}

	if err := api.do(request, &res); err != nil {
		return SeedStatusInvalid, err
	}
	log.Printf("debug: API seed %s status: %d (%d%%)", id, res.Status, res.Progress)

	if !res.Status.IsValid() {
		log.Printf("error: API returned invalid seed status: %d", res.Status)
	}

	return res.Status, nil
}

// GetSeedSpoilerLog returns the raw JSON spoiler log for a generated seed.
func (api *API) GetSeedSpoilerLog(id string) ([]byte, error) {
	log.Printf("debug: fetching API seed spoiler log for ID  %s", id)

	url := api.getURL("/seed/details", url.Values{"id": {id}})
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var res struct {
		SpoilerLog string `json:"spoilerLog"`
	}
	if err := api.do(request, &res); err != nil {
		return nil, err
	}

	return []byte(res.SpoilerLog), nil
}

// GetSeedPatch returns the raw binary patch that can be used to generate an OOTR rom.
func (api *API) GetSeedPatch(id string) ([]byte, error) {
	log.Printf("debug: fetching API seed patch for ID  %s", id)

	url := api.getURL("/seed/patch", url.Values{"id": {id}})

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var ret []byte
	if err := api.do(request, &ret); err != nil {
		return nil, err
	}

	return ret, nil
}

// UnlockSeedSpoilerLog allows access to the seed spoiler log from the OOTR
// website. This is only useful for seeds generated with the API-specific
// 'locked' parameter.
func (api *API) UnlockSeedSpoilerLog(id string) error {
	log.Printf("debug: unlocking API seed spoiler logs for ID  %s", id)

	url := api.getURL("/seed/unlock", url.Values{"id": {id}})
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	return api.do(request, nil)
}

var errRateLimit = errors.New("triggered API rate-limiter")

// do performs a rate-limited request on the API and writes the JSON-decoded
// response body in response.
// If response is nil the body is discarded, if response is *[]byte, the body
// is written raw.
func (api *API) do(request *http.Request, response interface{}) error {
	var tries int

	for {
		err := api.doInner(request, response)
		if errors.Is(err, errRateLimit) {
			tries++
			log.Printf("warning: rate-limited %d times", tries)
			continue
		}

		return err
	}
}

func (api *API) doInner(request *http.Request, response interface{}) error {
	start := time.Now()
	if err := api.limiter.Wait(context.TODO()); err != nil {
		return err
	}
	log.Printf("debug: waited %s before calling API", time.Since(start))

	res, err := api.http.Do(request)
	if err != nil {
		return fmt.Errorf("unable to perform HTTP request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusTooManyRequests {
		return errRateLimit
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("got status code %d", res.StatusCode)
	}

	// Special case, treat byte slices as asking for raw contents
	if response, ok := response.(*[]byte); ok {
		*response, err = ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}

		return nil
	}

	if response == nil {
		return nil
	}

	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&response); err != nil {
		return fmt.Errorf("unable to parse response: %s", err)
	}

	return nil
}
