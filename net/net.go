package net

import (
	"net"
	"net/url"
	"strings"
)

func isNetworkError(_err error) bool {
	if _err == nil {
		return false
	}
	switch t := _err.(type) {
	case net.Error:
		if t.Timeout() {
			return true
		}
		return true
	case *url.Error:
		if nestErr, ok := t.Err.(net.Error); ok {
			if nestErr.Timeout() {
				return true
			}
			return true
		}
	}
	if _err != nil && strings.Contains(_err.Error(), "use of closed network connection") {
		return true
	}
	return false
}
