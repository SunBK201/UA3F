package redirect

import (
	"fmt"
	"log/slog"

	"github.com/dlclark/regexp2"
	"github.com/sunbk201/ua3f/internal/common"
)

type Redirect307 struct {
	regex        *regexp2.Regexp
	replaceValue string
}

func (r *Redirect307) Type() common.ActionType {
	return common.ActionRedirect307
}

func (r *Redirect307) Execute(metadata *common.Metadata) (bool, error) {
	url := metadata.URL()

	if match, _ := r.regex.MatchString(url); !match {
		return true, nil
	}

	url, err := r.regex.Replace(url, r.replaceValue, -1, -1)
	if err != nil {
		slog.Error("Redirect307 r.regex.Replace", "error", err)
		return false, err
	}

	response := fmt.Sprintf("HTTP/1.1 307 Temporary Redirect\r\n"+
		"Location: %s\r\n"+
		"\r\n", url)

	_, err = metadata.ConnLink.LConn.Write([]byte(response))

	return false, err
}

func (r *Redirect307) Direction() common.Direction {
	return common.DirectionRequest
}

func (r *Redirect307) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(r.Type())),
		slog.String("regex", r.regex.String()),
		slog.String("replace_value", r.replaceValue),
	)
}

func NewRedirect307(regex string, replaceValue string) *Redirect307 {
	compiledRegex, err := regexp2.Compile(regex, regexp2.None)
	if err != nil {
		slog.Error("Failed to compile regex for Redirect307", "error", err)
		return nil
	}

	return &Redirect307{
		regex:        compiledRegex,
		replaceValue: replaceValue,
	}
}
