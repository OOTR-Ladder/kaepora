package config_test

import (
	"kaepora/internal/config"
	"net/url"
	"testing"
	"time"
)

func TestSignURLBadWebToken(t *testing.T) {
	c := config.Config{WebToken: ""}
	if _, err := c.SignURL("", time.Duration(0)); err == nil {
		t.Error("expected error on empty HMAC key")
	}
}

func TestSignURL(t *testing.T) {
	c := config.Config{WebToken: "00000000000000000000000000000000"}
	uris := []string{
		"https://ootrandomizer.com",
		"http://ootrandomizer.com/path?a=foo&y=baz&z=bar",
		"https://ootrandomizer.com/path?z=bar&a=foo&y=baz&t=bad",
	}

	for _, v := range uris {
		str, err := c.SignURL(v, 1*time.Hour)
		if err != nil {
			t.Error(err)
			continue
		}

		if err := c.CheckURL(str); err != nil {
			t.Error(err)
		}
	}
}

func TestSignURLExpired(t *testing.T) {
	c := config.Config{WebToken: "00000000000000000000000000000000"}
	str, err := c.SignURL("https://ootrandomizer.com", -1*time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.CheckURL(str); err == nil {
		t.Fatal("expected expired token")
	}
}

func TestSignURLBadToken(t *testing.T) {
	c := config.Config{WebToken: "00000000000000000000000000000000"}
	str, err := c.SignURL("https://ootrandomizer.com", 1*time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	u, _ := url.Parse(str)
	q := u.Query()
	q.Set("t", "invalid")
	u.RawQuery = q.Encode()
	str = u.String()

	if err := c.CheckURL(str); err == nil {
		t.Fatal("expected bad token")
	}
}
