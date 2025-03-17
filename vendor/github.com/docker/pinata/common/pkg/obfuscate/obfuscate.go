package obfuscate

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/docker/pinata/common/pkg/stringutil"
	"mvdan.cc/xurls/v2"
)

const (
	ObfuscatedHome = "<HOME>"
	ObfuscatedUser = "<USER>"
	ObfuscatedHost = "<HOST>"
)

var (
	protectedQueryParameters = []string{
		strings.ToUpper("client_id"),
		strings.ToUpper("code_challenge"),
		strings.ToUpper("state"),
		strings.ToUpper("login"),
		strings.ToUpper("password"),
		strings.ToUpper("token"),
		strings.ToUpper("access_token"),
		strings.ToUpper("refresh_token"),
	}
	userCurrentOnce    sync.Once
	loginName          = ""
	userName           = ""
	homeDir            = ""
	hostName           = ""
	hostNameUpperCased = ""
	keyRegExp          = regexp.MustCompile("(-----BEGIN [^-]*? KEY-----)(?s:.*?)(-----END [^-]*? KEY-----)")
	certRegExp         = regexp.MustCompile("(-----BEGIN CERTIFICATE-----)(?s:.*?)(-----END CERTIFICATE-----)")
)

// OverrideUsernameForTests is used in tests to temporary changes the username and loginName.
// It returns a function that restore the username and loginName to its previous state.
func OverrideUsernameForTests(tmpUsername string) func() {
	ObfuscateString("")
	previousUserName := userName
	userName = tmpUsername
	previousLoginName := loginName
	loginName = tmpUsername
	return func() { userName = previousUserName; loginName = previousLoginName }
}

func ObfuscateString(in string) string {
	userCurrentOnce.Do(func() {
		usr, err := user.Current()
		if err != nil {
			return
		}
		loginName = usr.Username
		userName = usr.Name
		homeDir = usr.HomeDir
		hostName, err = os.Hostname()
		if err != nil {
			return
		}
		hostNameUpperCased = strings.ToUpper(hostName)
	})

	if stringutil.HasPrefix(in, []string{"https://", "http://", "docker-desktop://"}) {
		in = URLFromString(in)
	}

	in = replaceString(in, homeDir, ObfuscatedHome)
	in = replaceString(in, loginName, ObfuscatedUser)
	in = replaceString(in, userName, ObfuscatedUser)
	in = replaceString(in, hostName, ObfuscatedHost)
	in = replaceString(in, hostNameUpperCased, ObfuscatedHost)
	in = obfuscateKeys(in)
	in = obfuscateCertificates(in)
	return in
}

// JsonString will do obfuscation on object and create a json string output
func JsonString(in any) string {
	switch in := in.(type) {
	case map[string]any:
		return mapToJsonString(in)
	default:
		bytes, err := json.Marshal(in)
		if err != nil {
			return fmt.Sprintf("obfuscate failed to marshal %+v: %v", in, err)
		}
		if bytes[0] != '{' {
			return fmt.Sprintf("%+v", in)
		}
		var mapValue map[string]any
		if err := json.Unmarshal(bytes, &mapValue); err != nil {
			return fmt.Sprintf("obfuscate failed to unmarshal %s: %v", string(bytes), err)
		}
		return mapToJsonString(mapValue)
	}
}

func mapToJsonString(in map[string]any) string {
	out := SensitiveValue(in)
	s, err := json.Marshal(out)
	if err != nil {
		return fmt.Sprintf("obfuscate failed to marshal %+v: %v", out, err)
	}
	return string(s)
}

// SensitiveValue will replace some part of the values only (like username, homedir, ips)
func SensitiveValue(in any) any {
	switch in := in.(type) {
	case string:
		return ObfuscateString(in)
	case []any:
		out := make([]any, len(in))
		for i, v := range in {
			out[i] = SensitiveValue(v)
		}
		return out
	case []string:
		out := make([]string, len(in))
		for i, v := range in {
			out[i] = ObfuscateString(v)
		}
		return out
	case map[any]any:
		out := make(map[any]any, len(in))
		for k, v := range in {
			out[k] = ObfuscateKV(k, v)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(in))
		for k, v := range in {
			out[k] = ObfuscateKV(k, v)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(in))
		for k, v := range in {
			out[k] = ObfuscateKV(k, v).(string)
		}
		return out
	default:
		return in
	}
}

var (
	// protectedKeys is an array including strings that, when part of a key
	// in a key-value data representation (a map), indicate the corresponding value may
	// be sensitive and should be obfuscated.
	//
	// Only used in [ObfuscateKV]
	protectedKeys = []string{
		"id",
		"token",
		"user",
		"username",
		"password",
		"certificates",
		"authorities",
		"http",
		"https",
	}
	// full string matches that should be *not* obfuscated
	// even if they contain one of the [protectedKeys]
	//
	// Only used in [ObfuscateKV]
	excludedKeys = map[string]struct{}{
		"idle": {},
	}
)

// ObfuscateKV is a function that obfuscates sensitive values in a key-value data representation.
//
// Based on the content of the key `k` and the list of [protectedKeys] and [excludedKeys],
// the function decides whether the  value `v` should be obfuscated.
func ObfuscateKV(k, v any) any {
	// keep harmless types
	switch v := v.(type) {
	case bool:
		return v
	case int:
		return v
	case float64:
		return v
	default:
	}

	// k is protected key, it should be obfuscated
	if k, ok := k.(string); ok {
		k = strings.ToLower(k)
		if _, ok := excludedKeys[k]; !ok {
			if stringutil.Contains(k, protectedKeys) {
				return obfuscateValue(v)
			}
		}
	}

	return SensitiveValue(v)
}

func obfuscateValue(in any) string {
	switch in := in.(type) {
	case string:
		return obfuscateProtectedKeyString(in)
	default:
		if in == nil {
			return "null"
		}
		return reflect.TypeOf(in).String()
	}
}

func obfuscateProtectedKeyString(in string) string {
	if in == "" {
		return ""
	}

	chars := []rune(in)
	inLen := len(chars)
	if inLen < 5 {
		return "..."
	}

	var out strings.Builder
	out.Grow(16)
	out.WriteString("...")
	return out.String()
}

func replaceString(in, old, new string) string {
	if old == "" || old == "docker" || len(old) < 3 {
		return in
	}
	return strings.ReplaceAll(in, old, new)
}

func mustCompileStrictMatcher() *regexp.Regexp {
	r, err := xurls.StrictMatchingScheme(xurls.AnyScheme)
	if err != nil {
		panic(err)
	}
	return r
}

var strictMatcher = mustCompileStrictMatcher()

// URLFromString obfuscates the URL by hiding User information (login and password) and obfuscating Query parameters
func URLFromString(in string) string {
	return strictMatcher.ReplaceAllStringFunc(in, func(s string) string {
		if s == "" {
			return s
		}
		u, err := url.Parse(s)
		if err != nil {
			return s
		}
		tempUrl := URL(*u)
		return tempUrl
	})
}

// URL obfuscates the URL by hiding User information (login and password) and obfuscating Query parameters
func URL(u url.URL) string {
	// Hide User information (login and password) from URL if present
	if u.User.Username() != "" {
		u.User = url.User("USERINFO")
	}

	// Obfuscate Query parameters
	qp := u.Query()
	for k := range qp {
		// For each Query parameter, obfuscate if it is a protected key
		for _, p := range protectedQueryParameters {
			if strings.Contains(p, strings.ToUpper(k)) {
				qp.Set(k, "REDACTED")
			}
		}
		// Obfuscate the value of the Query parameter if it's a URL, even if the key is not protected
		if strings.HasPrefix(qp.Get(k), "https") {
			// decode url Query Parameter
			decoded, _ := url.QueryUnescape(qp.Get(k))
			qp.Set(k, URLFromString(decoded))
		}
	}
	// Write Query parameters back to the URL
	u.RawQuery = qp.Encode()
	return u.String()
}

func obfuscateKeys(in string) string {
	return keyRegExp.ReplaceAllString(in, "$1<REDACTED>$2")
}

func obfuscateCertificates(in string) string {
	return certRegExp.ReplaceAllString(in, "$1<REDACTED>$2")
}
