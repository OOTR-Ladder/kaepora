package config

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// ErrTokenExpired means the token is valid, but expired.
var ErrTokenExpired = errors.New("token expired")

// SignURL adds a signed token query parameter to an URL, valid for the given duration.
func (c *Config) SignURL(str string, d time.Duration) (string, error) {
	u, err := url.Parse(str)
	if err != nil {
		return "", fmt.Errorf("unable to parse URL: %w", err)
	}

	u.Scheme = "https"
	q := u.Query()
	q.Del("t")
	q.Set("td", strconv.FormatInt(time.Now().Add(d).Unix(), 10))
	u.RawQuery = q.Encode()

	token, err := c.sign([]byte(u.String()))
	if err != nil {
		return "", err
	}

	q.Set("t", token)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// CheckURL ensures the given URL is properly signed.
func (c *Config) CheckURL(str string) error {
	u, err := url.Parse(str)
	if err != nil {
		return fmt.Errorf("unable to parse URL: %w", err)
	}

	u.Scheme = "https"
	q := u.Query()
	td, err := strconv.ParseInt(q.Get("td"), 10, 64)
	if err != nil {
		return err
	}

	inputToken := q.Get("t")
	q.Del("t")
	u.RawQuery = q.Encode()

	token, err := c.sign([]byte(u.String()))
	if err != nil {
		return err
	}

	if token != inputToken {
		return errors.New("invalid token")
	}

	// Keep this last, this error must be returned _only_ if the token is valid.
	if time.Unix(td, 0).Before(time.Now()) {
		return ErrTokenExpired
	}

	return nil
}

func (c *Config) sign(b []byte) (string, error) {
	if len(c.WebToken) < 32 {
		return "", errors.New("web token must be ≥ 32 chars")
	}

	mac := hmac.New(sha256.New, []byte(c.WebToken))
	if _, err := mac.Write(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(mac.Sum(nil)), nil
}
