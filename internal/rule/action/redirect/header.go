package redirect

import (
	"encoding/json"
	"log/slog"
	"net/url"

	"github.com/dlclark/regexp2"
	"github.com/sunbk201/ua3f/internal/common"
)

type RedirectHeader struct {
	regex        *regexp2.Regexp
	replaceValue string
}

func (r *RedirectHeader) Type() common.ActionType {
	return common.ActionRedirectHeader
}

func (r *RedirectHeader) Execute(metadata *common.Metadata) (bool, error) {
	requestUrl := metadata.URL()

	if match, _ := r.regex.MatchString(requestUrl); !match {
		return true, nil
	}

	requestUrl, err := r.regex.Replace(requestUrl, r.replaceValue, -1, -1)
	if err != nil {
		slog.Error("RedirectHeader r.regex.Replace", "error", err)
		return false, err
	}

	u, err := url.Parse(requestUrl)
	if err != nil {
		slog.Error("RedirectHeader url.Parse", "error", err)
		return false, err
	}

	if metadata.Request.Host == u.Host {
		metadata.Request.URL = u
		return true, nil
	}

	clientReq := metadata.Request.Clone(metadata.Request.Context())
	clientReq.URL = u
	clientReq.RequestURI = ""
	clientReq.Host = u.Host
	clientReq.Header.Set("Host", u.Host)
	resp, err := sendRequest(clientReq)
	if err != nil {
		slog.Error("RedirectHeader sendRequest", "error", err)
		return false, err
	}
	resp.Header.Set("Connection", "close")
	err = resp.Write(metadata.ConnLink.LConn)
	if err != nil {
		slog.Error("RedirectHeader resp.Write", "error", err)
	}

	return false, err
}

func (r *RedirectHeader) Direction() common.Direction {
	return common.DirectionRequest
}

func (r *RedirectHeader) MarshalJSON() ([]byte, error) {
	var regex string
	if r.regex != nil {
		regex = r.regex.String()
	}
	return json.Marshal(map[string]any{
		"type":          r.Type(),
		"regex":         regex,
		"replace_value": r.replaceValue,
	})
}

func (r *RedirectHeader) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(r.Type())),
		slog.String("regex", r.regex.String()),
		slog.String("replace_value", r.replaceValue),
	)
}

func NewRedirectHeader(regex string, replaceValue string) *RedirectHeader {
	compiledRegex, err := regexp2.Compile(regex, regexp2.None)
	if err != nil {
		slog.Error("Failed to compile regex for RedirectHeader", "error", err)
		return nil
	}

	return &RedirectHeader{
		regex:        compiledRegex,
		replaceValue: replaceValue,
	}
}
