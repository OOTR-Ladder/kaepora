package util

import (
	"fmt"
	"log"
	"net/url"
	"strings"
)

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
	u.Host = strings.ToLower(u.Host)
	u.RawQuery = ""
	u.Fragment = ""
	if !isStreamHostAllowed(u.Host) {
		return "", ErrPublic(fmt.Sprintf("%s is not a known streaming platform", u.Host))
	}

	if u.Path == "" || u.Path == "/" {
		return "", ErrPublic(fmt.Sprintf("you can't have an empty %s username", u.Host))
	}

	return u.String(), nil
}

func isStreamHostAllowed(host string) bool {
	allowed := []string{
		"www.twitch.tv",
		"mixer.com",
	}
	for _, v := range allowed {
		if host == v {
			return true
		}
	}

	return false
}
