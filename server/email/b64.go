package email

import "encoding/base64"

func stdB64(s string) ([]byte, error)    { return base64.StdEncoding.DecodeString(s) }
func urlB64(s string) ([]byte, error)    { return base64.URLEncoding.DecodeString(s) }
func stdRawB64(s string) ([]byte, error) { return base64.RawStdEncoding.DecodeString(s) }
func urlRawB64(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) }
