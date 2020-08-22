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
	str, err := c.SignURL("https://ootrandomizer.com?u=Gaebora", 1*time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.CheckURL(str); err != nil {
		t.Fatal(err)
	}
}

func TestSignURLOverride(t *testing.T) {
	c := config.Config{WebToken: "00000000000000000000000000000000"}
	str, err := c.SignURL("https://ootrandomizer.com?t=foo&td=42", 1*time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.CheckURL(str); err != nil {
		t.Fatal(err)
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
