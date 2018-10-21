package quicklog

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
)

type Config struct {
	ProjectID int
	Source    string
	ApiKey    string
	ApiURL    string
	Client    *http.Client
}

type Ctx struct {
	ActorID      string
	TraceID      string
	ParentSpanID string
	SpanID       string
}

type entryBody struct {
	ProjectID    int         `json:"project_id"`
	Published    time.Time   `json:"published"`
	Source       string      `json:"source"`
	Actor        string      `json:"actor"`
	Type         string      `json:"type"`
	Object       string      `json:"object"`
	Target       string      `json:"target"`
	Context      interface{} `json:"context"`
	TraceID      string      `json:"trace_id"`
	ParentSpanID string      `json:"parent_span_id"`
	SpanID       string      `json:"span_id"`
}

type tagBody struct {
	ProjectID int    `json:"project_id"`
	TraceID   string `json:"trace_id"`
	Tag       string `json:"tag"`
}

var (
	config Config
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func Configure(c Config) {
	config = c
	if config.ApiURL == "" {
		config.ApiURL = "https://api.quicklog.io"
	}
	if config.Client == nil {
		tr := http.Transport{
			MaxIdleConns:       5,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: true,
		}
		config.Client = &http.Client{Transport: &tr, Timeout: 3 * time.Second}
	}
}

/**
 * Creates a quicklog entry.
 * @param {action} a type or other identifying event name
 * @param {object} identifier of primary 'thing' (often formatted as kind:unique-id)
 * @param {target} identifier of secondary 'thing' (sometimes a destination)
 * @param {extra} other useful information with string keys and JSON serializable values
 * @param {traceCtx}
 * @param {tags} e.g. ["name:value", "value", "name:value:containing:colons", ":value:containing:colons" ]
 * @return error
 */
func Quicklog(published time.Time, action, object, target string, extra map[string]interface{}, traceCtx Ctx, tags ...string) error {
	if config.ProjectID == 0 {
		return fmt.Errorf("ProjectID must be set in Config options")
	}
	if config.ApiKey == "" {
		return fmt.Errorf("ApiKey must be set in Config options")
	}
	if config.ApiURL == "" {
		return fmt.Errorf("ApiURL must be set in Config options")
	}

	url := config.ApiURL + "/entries?api_key=" + config.ApiKey

	body := entryBody{
		ProjectID:    config.ProjectID,
		Published:    published,
		Source:       config.Source,
		Actor:        traceCtx.ActorID,
		Type:         action,
		Object:       object,
		Target:       target,
		Context:      extra,
		TraceID:      traceCtx.TraceID,
		ParentSpanID: traceCtx.ParentSpanID,
		SpanID:       traceCtx.SpanID,
	}

	content, err := json.Marshal(body)
	if err != nil {
		return err
	}

	resp, err := config.Client.Post(url, "application/json", bytes.NewReader(content))
	defer resp.Body.Close()
	if err != nil {
		errBody, err2 := ioutil.ReadAll(resp.Body)
		if len(errBody) != 0 && err2 == nil {
			return fmt.Errorf("%v : BODY = %s", err.Error(), string(errBody))
		} else {
			return err
		}
	}

	err = TagTrace(traceCtx.TraceID, tags...)
	return err
}

/**
 * Associates a tag (e.g key:value) with the current trace.
 * @param {string} tag (format 'key:value' or 'value', or ':value:containing-colon')
 * @param {object} traceOpts ('actorId', 'traceId', 'parentSpanId', and 'spanId' used from request to response)
 * @return {promise} axios.post()
 */
func TagTrace(traceID string, tags ...string) error {
	if len(tags) == 0 {
		return nil
	}
	if config.ProjectID == 0 {
		return fmt.Errorf("ProjectId must be set in Config options")
	}
	if config.ApiKey == "" {
		return fmt.Errorf("ApiKey must be set in Config options")
	}
	if config.ApiURL == "" {
		return fmt.Errorf("ApiURL must be set in Config options")
	}
	if traceID == "" {
		return fmt.Errorf("'traceID' must be a non-empty string")
	}

	url := config.ApiURL + "/tags?api_key=" + config.ApiKey

	body := tagBody{
		ProjectID: config.ProjectID,
		TraceID:   traceID,
	}

	emptyTag := false
	for _, tag := range tags {
		if tag == "" {
			emptyTag = true
			continue
		}

		body.Tag = tag
		content, err := json.Marshal(body)
		if err != nil {
			return err
		}

		resp, err := config.Client.Post(url, "application/json", bytes.NewReader(content))
		defer resp.Body.Close()
		if err != nil {
			errBody, err2 := ioutil.ReadAll(resp.Body)
			if len(errBody) != 0 && err2 == nil {
				return fmt.Errorf("%v : BODY = %s", err.Error(), string(errBody))
			} else {
				return err
			}
		}
	}
	if emptyTag {
		return fmt.Errorf("'tags' must contain non-empty strings")
	}
	return nil
}

/**
 * Creates a Ctx containing 'ActorID', 'TraceID', 'ParentSpanID', and a newly generated 'SpanID'.
 * If called with an empty 'traceID', it is set to the new SpanID, and ParentSpanID will be empty.
 * @param {string} actorID
 * @param {string} traceID
 * @param {string} parentSpanID
 */
func TraceCtx(actorID, traceID, parentSpanID string) Ctx {
	spanID := GenerateID()
	if traceID == "" {
		traceID = spanID
		parentSpanID = ""
	}
	return Ctx{
		ActorID:      actorID,
		TraceID:      traceID,
		ParentSpanID: parentSpanID,
		SpanID:       spanID,
	}
}

func GenerateID() string {
	src := make([]byte, 8)
	binary.LittleEndian.PutUint64(src, rand.Uint64())
	dst := make([]byte, hex.EncodedLen(len(src)))

	hex.Encode(dst, src)
	return string(dst)
}
