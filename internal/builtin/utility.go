package builtin

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/anthropics/slop/internal/evaluator"
)

func (r *Registry) registerUtilityFunctions() {
	// Time
	r.Register("now", builtinNow)
	r.Register("today", builtinToday)
	r.Register("time_parse", builtinTimeParse)
	r.Register("time_format", builtinTimeFormat)
	r.Register("time_add", builtinTimeAdd)
	r.Register("time_diff", builtinTimeDiff)
	r.Register("sleep", builtinSleep)

	// JSON
	r.Register("json_parse", builtinJsonParse)
	r.Register("json_stringify", builtinJsonStringify)

	// Encoding
	r.Register("base64_encode", builtinBase64Encode)
	r.Register("base64_decode", builtinBase64Decode)
	r.Register("url_encode", builtinUrlEncode)
	r.Register("url_decode", builtinUrlDecode)
	r.Register("html_escape", builtinHtmlEscape)
	r.Register("html_unescape", builtinHtmlUnescape)

	// Hashing
	r.Register("hash_md5", builtinHashMd5)
	r.Register("hash_sha256", builtinHashSha256)
	r.Register("hash_sha512", builtinHashSha512)
	r.Register("hash_hmac", builtinHashHmac)

	// Regex
	r.Register("regex_match", builtinRegexMatch)
	r.Register("regex_find_all", builtinRegexFindAll)
	r.Register("regex_replace", builtinRegexReplace)
	r.Register("regex_split", builtinRegexSplit)
	r.Register("regex_test", builtinRegexTest)

	// Validation
	r.Register("validate_email", builtinValidateEmail)
	r.Register("validate_url", builtinValidateUrl)
	r.Register("validate_uuid", builtinValidateUuid)
	r.Register("validate_json", builtinValidateJson)

	// UUID
	r.Register("uuid", builtinUuid)

	// Template matching
	r.Register("match", builtinMatch)
}

// Time functions

func builtinNow(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("now", args, 0); err != nil {
		return nil, err
	}
	return &evaluator.StringValue{Value: time.Now().UTC().Format(time.RFC3339)}, nil
}

func builtinToday(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("today", args, 0); err != nil {
		return nil, err
	}
	return &evaluator.StringValue{Value: time.Now().UTC().Format("2006-01-02")}, nil
}

func builtinTimeParse(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("time_parse", args, 1, 2); err != nil {
		return nil, err
	}

	s, err := requireString("time_parse", args[0])
	if err != nil {
		return nil, err
	}

	layout := time.RFC3339
	if len(args) == 2 {
		layout, err = requireString("time_parse", args[1])
		if err != nil {
			return nil, err
		}
	}

	t, err := time.Parse(layout, s)
	if err != nil {
		return nil, fmt.Errorf("time_parse() invalid time format: %s", err)
	}

	return &evaluator.StringValue{Value: t.UTC().Format(time.RFC3339)}, nil
}

func builtinTimeFormat(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("time_format", args, 2); err != nil {
		return nil, err
	}

	ts, err := requireString("time_format", args[0])
	if err != nil {
		return nil, err
	}

	format, err := requireString("time_format", args[1])
	if err != nil {
		return nil, err
	}

	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return nil, fmt.Errorf("time_format() invalid timestamp: %s", err)
	}

	return &evaluator.StringValue{Value: t.Format(format)}, nil
}

func builtinTimeAdd(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("time_add", args, 2); err != nil {
		return nil, err
	}

	ts, err := requireString("time_add", args[0])
	if err != nil {
		return nil, err
	}

	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return nil, fmt.Errorf("time_add() invalid timestamp: %s", err)
	}

	durationStr, err := requireString("time_add", args[1])
	if err != nil {
		return nil, err
	}

	d, err := time.ParseDuration(durationStr)
	if err != nil {
		return nil, fmt.Errorf("time_add() invalid duration: %s", err)
	}

	return &evaluator.StringValue{Value: t.Add(d).UTC().Format(time.RFC3339)}, nil
}

func builtinTimeDiff(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("time_diff", args, 2); err != nil {
		return nil, err
	}

	ts1, err := requireString("time_diff", args[0])
	if err != nil {
		return nil, err
	}

	ts2, err := requireString("time_diff", args[1])
	if err != nil {
		return nil, err
	}

	t1, err := time.Parse(time.RFC3339, ts1)
	if err != nil {
		return nil, fmt.Errorf("time_diff() invalid timestamp: %s", err)
	}

	t2, err := time.Parse(time.RFC3339, ts2)
	if err != nil {
		return nil, fmt.Errorf("time_diff() invalid timestamp: %s", err)
	}

	return &evaluator.FloatValue{Value: t1.Sub(t2).Seconds()}, nil
}

func builtinSleep(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("sleep", args, 1); err != nil {
		return nil, err
	}

	seconds, err := requireFloat("sleep", args[0])
	if err != nil {
		return nil, err
	}

	if seconds < 0 {
		return nil, fmt.Errorf("sleep() duration must be non-negative")
	}

	time.Sleep(time.Duration(seconds * float64(time.Second)))
	return evaluator.NONE, nil
}

// JSON functions

func builtinJsonParse(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("json_parse", args, 1); err != nil {
		return nil, err
	}

	s, err := requireString("json_parse", args[0])
	if err != nil {
		return nil, err
	}

	var data interface{}
	if err := json.Unmarshal([]byte(s), &data); err != nil {
		return nil, fmt.Errorf("json_parse() invalid JSON: %s", err)
	}

	return jsonToValue(data), nil
}

func jsonToValue(data interface{}) evaluator.Value {
	switch v := data.(type) {
	case nil:
		return evaluator.NONE
	case bool:
		if v {
			return evaluator.TRUE
		}
		return evaluator.FALSE
	case float64:
		// Check if it's actually an integer
		if v == float64(int64(v)) {
			return &evaluator.IntValue{Value: int64(v)}
		}
		return &evaluator.FloatValue{Value: v}
	case string:
		return &evaluator.StringValue{Value: v}
	case []interface{}:
		items := make([]evaluator.Value, len(v))
		for i, item := range v {
			items[i] = jsonToValue(item)
		}
		return &evaluator.ListValue{Elements: items}
	case map[string]interface{}:
		pairs := make(map[string]evaluator.Value)
		for k, val := range v {
			pairs[k] = jsonToValue(val)
		}
		return &evaluator.MapValue{Pairs: pairs}
	default:
		return &evaluator.StringValue{Value: fmt.Sprintf("%v", v)}
	}
}

func builtinJsonStringify(args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("json_stringify", args, 1); err != nil {
		return nil, err
	}

	data := valueToJson(args[0])

	var bytes []byte
	var err error

	indent := 0
	if iv, ok := kwargs["indent"]; ok {
		if intVal, ok := iv.(*evaluator.IntValue); ok {
			indent = int(intVal.Value)
		}
	}

	if indent > 0 {
		bytes, err = json.MarshalIndent(data, "", strings.Repeat(" ", indent))
	} else {
		bytes, err = json.Marshal(data)
	}

	if err != nil {
		return nil, fmt.Errorf("json_stringify() error: %s", err)
	}

	return &evaluator.StringValue{Value: string(bytes)}, nil
}

func valueToJson(val evaluator.Value) interface{} {
	switch v := val.(type) {
	case *evaluator.NoneValue:
		return nil
	case *evaluator.BoolValue:
		return v.Value
	case *evaluator.IntValue:
		return v.Value
	case *evaluator.FloatValue:
		return v.Value
	case *evaluator.StringValue:
		return v.Value
	case *evaluator.ListValue:
		items := make([]interface{}, len(v.Elements))
		for i, item := range v.Elements {
			items[i] = valueToJson(item)
		}
		return items
	case *evaluator.MapValue:
		result := make(map[string]interface{})
		for k, val := range v.Pairs {
			result[k] = valueToJson(val)
		}
		return result
	default:
		return v.String()
	}
}

// Encoding functions

func builtinBase64Encode(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("base64_encode", args, 1); err != nil {
		return nil, err
	}

	s, err := requireString("base64_encode", args[0])
	if err != nil {
		return nil, err
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(s))
	return &evaluator.StringValue{Value: encoded}, nil
}

func builtinBase64Decode(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("base64_decode", args, 1); err != nil {
		return nil, err
	}

	s, err := requireString("base64_decode", args[0])
	if err != nil {
		return nil, err
	}

	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("base64_decode() invalid base64: %s", err)
	}

	return &evaluator.StringValue{Value: string(decoded)}, nil
}

func builtinUrlEncode(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("url_encode", args, 1); err != nil {
		return nil, err
	}

	s, err := requireString("url_encode", args[0])
	if err != nil {
		return nil, err
	}

	return &evaluator.StringValue{Value: url.QueryEscape(s)}, nil
}

func builtinUrlDecode(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("url_decode", args, 1); err != nil {
		return nil, err
	}

	s, err := requireString("url_decode", args[0])
	if err != nil {
		return nil, err
	}

	decoded, err := url.QueryUnescape(s)
	if err != nil {
		return nil, fmt.Errorf("url_decode() invalid URL encoding: %s", err)
	}

	return &evaluator.StringValue{Value: decoded}, nil
}

func builtinHtmlEscape(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("html_escape", args, 1); err != nil {
		return nil, err
	}

	s, err := requireString("html_escape", args[0])
	if err != nil {
		return nil, err
	}

	return &evaluator.StringValue{Value: html.EscapeString(s)}, nil
}

func builtinHtmlUnescape(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("html_unescape", args, 1); err != nil {
		return nil, err
	}

	s, err := requireString("html_unescape", args[0])
	if err != nil {
		return nil, err
	}

	return &evaluator.StringValue{Value: html.UnescapeString(s)}, nil
}

// Hashing functions

func builtinHashMd5(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("hash_md5", args, 1); err != nil {
		return nil, err
	}

	s, err := requireString("hash_md5", args[0])
	if err != nil {
		return nil, err
	}

	hash := md5.Sum([]byte(s))
	return &evaluator.StringValue{Value: hex.EncodeToString(hash[:])}, nil
}

func builtinHashSha256(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("hash_sha256", args, 1); err != nil {
		return nil, err
	}

	s, err := requireString("hash_sha256", args[0])
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256([]byte(s))
	return &evaluator.StringValue{Value: hex.EncodeToString(hash[:])}, nil
}

func builtinHashSha512(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("hash_sha512", args, 1); err != nil {
		return nil, err
	}

	s, err := requireString("hash_sha512", args[0])
	if err != nil {
		return nil, err
	}

	hash := sha512.Sum512([]byte(s))
	return &evaluator.StringValue{Value: hex.EncodeToString(hash[:])}, nil
}

func builtinHashHmac(args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("hash_hmac", args, 2); err != nil {
		return nil, err
	}

	s, err := requireString("hash_hmac", args[0])
	if err != nil {
		return nil, err
	}

	key, err := requireString("hash_hmac", args[1])
	if err != nil {
		return nil, err
	}

	algorithm := "sha256"
	if alg, ok := kwargs["algorithm"]; ok {
		if sv, ok := alg.(*evaluator.StringValue); ok {
			algorithm = sv.Value
		}
	}

	var hash []byte
	switch algorithm {
	case "sha256":
		h := hmac.New(sha256.New, []byte(key))
		h.Write([]byte(s))
		hash = h.Sum(nil)
	case "sha512":
		h := hmac.New(sha512.New, []byte(key))
		h.Write([]byte(s))
		hash = h.Sum(nil)
	case "md5":
		h := hmac.New(md5.New, []byte(key))
		h.Write([]byte(s))
		hash = h.Sum(nil)
	default:
		return nil, fmt.Errorf("hash_hmac() unsupported algorithm: %s", algorithm)
	}

	return &evaluator.StringValue{Value: hex.EncodeToString(hash)}, nil
}

// Regex functions

func builtinRegexMatch(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("regex_match", args, 2); err != nil {
		return nil, err
	}

	pattern, err := requireString("regex_match", args[0])
	if err != nil {
		return nil, err
	}

	s, err := requireString("regex_match", args[1])
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("regex_match() invalid pattern: %s", err)
	}

	match := re.FindString(s)
	if match == "" {
		return evaluator.NONE, nil
	}

	return &evaluator.StringValue{Value: match}, nil
}

func builtinRegexFindAll(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("regex_find_all", args, 2); err != nil {
		return nil, err
	}

	pattern, err := requireString("regex_find_all", args[0])
	if err != nil {
		return nil, err
	}

	s, err := requireString("regex_find_all", args[1])
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("regex_find_all() invalid pattern: %s", err)
	}

	matches := re.FindAllString(s, -1)
	items := make([]evaluator.Value, len(matches))
	for i, m := range matches {
		items[i] = &evaluator.StringValue{Value: m}
	}

	return &evaluator.ListValue{Elements: items}, nil
}

func builtinRegexReplace(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("regex_replace", args, 3); err != nil {
		return nil, err
	}

	pattern, err := requireString("regex_replace", args[0])
	if err != nil {
		return nil, err
	}

	s, err := requireString("regex_replace", args[1])
	if err != nil {
		return nil, err
	}

	replacement, err := requireString("regex_replace", args[2])
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("regex_replace() invalid pattern: %s", err)
	}

	result := re.ReplaceAllString(s, replacement)
	return &evaluator.StringValue{Value: result}, nil
}

func builtinRegexSplit(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("regex_split", args, 2); err != nil {
		return nil, err
	}

	pattern, err := requireString("regex_split", args[0])
	if err != nil {
		return nil, err
	}

	s, err := requireString("regex_split", args[1])
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("regex_split() invalid pattern: %s", err)
	}

	parts := re.Split(s, -1)
	items := make([]evaluator.Value, len(parts))
	for i, p := range parts {
		items[i] = &evaluator.StringValue{Value: p}
	}

	return &evaluator.ListValue{Elements: items}, nil
}

func builtinRegexTest(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("regex_test", args, 2); err != nil {
		return nil, err
	}

	pattern, err := requireString("regex_test", args[0])
	if err != nil {
		return nil, err
	}

	s, err := requireString("regex_test", args[1])
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("regex_test() invalid pattern: %s", err)
	}

	if re.MatchString(s) {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

// Validation functions

func builtinValidateEmail(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("validate_email", args, 1); err != nil {
		return nil, err
	}

	s, err := requireString("validate_email", args[0])
	if err != nil {
		return nil, err
	}

	_, err = mail.ParseAddress(s)
	if err != nil {
		return evaluator.FALSE, nil
	}
	return evaluator.TRUE, nil
}

func builtinValidateUrl(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("validate_url", args, 1); err != nil {
		return nil, err
	}

	s, err := requireString("validate_url", args[0])
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(s)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return evaluator.FALSE, nil
	}
	return evaluator.TRUE, nil
}

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func builtinValidateUuid(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("validate_uuid", args, 1); err != nil {
		return nil, err
	}

	s, err := requireString("validate_uuid", args[0])
	if err != nil {
		return nil, err
	}

	if uuidRegex.MatchString(s) {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

func builtinValidateJson(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("validate_json", args, 1); err != nil {
		return nil, err
	}

	s, err := requireString("validate_json", args[0])
	if err != nil {
		return nil, err
	}

	if json.Valid([]byte(s)) {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

// UUID function
func builtinUuid(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("uuid", args, 0); err != nil {
		return nil, err
	}

	// Generate a simple UUID v4 (random)
	// This is a simplified implementation - in production you'd use a proper UUID library
	b := make([]byte, 16)
	for i := range b {
		b[i] = byte(time.Now().UnixNano() ^ int64(i*31))
	}

	// Set version (4) and variant bits
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
	return &evaluator.StringValue{Value: uuid}, nil
}
