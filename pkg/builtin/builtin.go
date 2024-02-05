package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/acorn-io/gptscript/pkg/types"
)

var Tools = map[string]types.Tool{
	"sys.read": {
		Description: "Reads the contents of a file",
		Arguments: types.ObjectSchema(
			"filename", "The name of the file to read"),
		BuiltinFunc: SysRead,
	},
	"sys.write": {
		Description: "Write the contents to a file",
		Arguments: types.ObjectSchema(
			"filename", "The name of the file to write to",
			"content", "The content to write"),
		BuiltinFunc: SysWrite,
	},
	"sys.http.get": {
		Description: "Download the contents of a http or https URL",
		Arguments: types.ObjectSchema(
			"url", "The URL to download"),
		BuiltinFunc: SysHTTPGet,
	},
	"sys.abort": {
		Description: "Aborts execution",
		Arguments: types.ObjectSchema(
			"message", "The description of the error or unexpected result that caused abort to be called",
		),
		BuiltinFunc: SysAbort,
	},
	"sys.http.post": {
		Description: "Write contents to a http or https URL using the POST method",
		Arguments: types.ObjectSchema(
			"url", "The URL to POST to",
			"content", "The content to POST",
			"contentType", "The \"content type\" of the content such as application/json or text/plain"),
		BuiltinFunc: SysHTTPPost,
	},
}

func ListTools() (result []types.Tool) {
	var keys []string
	for k := range Tools {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	for _, key := range keys {
		t, _ := Builtin(key)
		result = append(result, t)
	}

	return
}

func Builtin(name string) (types.Tool, bool) {
	t, ok := Tools[name]
	t.Name = name
	t.ID = name
	t.Instructions = "#!" + name
	return t, ok
}

func SysRead(ctx context.Context, env []string, input string) (string, error) {
	var params struct {
		Filename string `json:"filename,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", err
	}

	log.Debugf("Reading file %s", params.Filename)
	data, err := os.ReadFile(params.Filename)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func SysWrite(ctx context.Context, env []string, input string) (string, error) {
	var params struct {
		Filename string `json:"filename,omitempty"`
		Content  string `json:"content,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", err
	}

	data := []byte(params.Content)
	msg := fmt.Sprintf("Wrote %d bytes to file %s", len(data), params.Filename)
	log.Debugf(msg)

	return "", os.WriteFile(params.Filename, data, 0644)
}

func SysHTTPGet(ctx context.Context, env []string, input string) (string, error) {
	var params struct {
		URL string `json:"url,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", err
	}

	log.Debugf("http get %s", params.URL)
	resp, err := http.Get(params.URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download %s: %s", params.URL, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func SysHTTPPost(ctx context.Context, env []string, input string) (string, error) {
	var params struct {
		URL         string `json:"url,omitempty"`
		Content     string `json:"content,omitempty"`
		ContentType string `json:"contentType,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, params.URL, strings.NewReader(params.Content))
	if err != nil {
		return "", err
	}
	if params.ContentType != "" {
		req.Header.Set("Content-Type", params.ContentType)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	_, _ = io.ReadAll(resp.Body)
	if resp.StatusCode > 399 {
		return "", fmt.Errorf("failed to post %s: %s", params.URL, resp.Status)
	}

	return fmt.Sprintf("Wrote %d to %s", len([]byte(params.Content)), params.URL), nil
}

func SysAbort(ctx context.Context, env []string, input string) (string, error) {
	var params struct {
		Message string `json:"message,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", err
	}
	return "", fmt.Errorf("ABORT: %s", params.Message)
}