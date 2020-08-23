package util

import (
	"fmt"
	"log"
	"net/url"
	"strings"
)

// NormalizeStreamURL returns the "canonical" URL for a given livestream URL.
func NormalizeStreamURL(str string) (string, error) {
	// Allow URL without schemes, without a scheme the host is parsed as part
	// of the path.
	if !strings.HasPrefix(str, "http") {
		str = "https://" + str
	}

	u, err := url.Parse(str)
	if err != nil {
		log.Printf("warning: %s", err)
		return "", ErrPublic("invalid URL")
	}

	u.Scheme = "https"
	u.RawQuery = ""
	u.Fragment = ""

	if u.Host, err = normalizeStreamHost(u.Host); err != nil {
		return "", err
	}

	if u.Path == "" || u.Path == "/" {
		return "", ErrPublic(fmt.Sprintf("you can't have an empty %s username", u.Host))
	}

	return u.String(), nil
}

func normalizeStreamHost(host string) (string, error) {
	switch host {
	case "www.twitch.tv", "twitch.tv":
		return "twitch.tv", nil
	}

	return "", ErrPublic(fmt.Sprintf("%s is not a known streaming platform", host))
}
