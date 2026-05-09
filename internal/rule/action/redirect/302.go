package redirect

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/dlclark/regexp2"
	"github.com/sunbk201/ua3f/internal/common"
)

type Redirect302 struct {
	regex        *regexp2.Regexp
	replaceValue string
}

func (r *Redirect302) Type() common.ActionType {
	return common.ActionRedirect302
}

func (r *Redirect302) Execute(metadata *common.Metadata) (bool, error) {
	url := metadata.URL()

	if match, _ := r.regex.MatchString(url); !match {
		return true, nil
	}

	url, err := r.regex.Replace(url, r.replaceValue, -1, -1)
	if err != nil {
		slog.Error("Redirect302 r.regex.Replace", "error", err)
		return false, err
	}

	response := fmt.Sprintf("HTTP/1.1 302 Found\r\n"+
		"Location: %s\r\n"+
		"\r\n", url)

	_, err = metadata.ConnLink.LConn.Write([]byte(response))

	return false, err
}

func (r *Redirect302) Direction() common.Direction {
	return common.DirectionRequest
}

func (r *Redirect302) MarshalJSON() ([]byte, error) {
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

func (r *Redirect302) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(r.Type())),
		slog.String("regex", r.regex.String()),
		slog.String("replace_value", r.replaceValue),
	)
}

func NewRedirect302(regex string, replaceValue string) *Redirect302 {
	compiledRegex, err := regexp2.Compile(regex, regexp2.None)
	if err != nil {
		slog.Error("Failed to compile regex for Redirect302", "error", err)
		return nil
	}

	return &Redirect302{
		regex:        compiledRegex,
		replaceValue: replaceValue,
	}
}
