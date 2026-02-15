package match

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rule/action"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type DomainSet struct {
	action    common.Action
	domainSet []string
	source    string // local file path or remote url
	mu        sync.RWMutex
	loaded    bool
}

func (d *DomainSet) Type() common.RuleType {
	return common.RuleTypeDomainSet
}

func (d *DomainSet) Match(metadata *common.Metadata) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, domain := range d.domainSet {
		if strings.HasSuffix(metadata.Host(), domain) {
			return true
		}
	}
	return false
}

func (d *DomainSet) Action() common.Action {
	return d.action
}

func (d *DomainSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":   d.Type(),
		"action": d.action,
	})
}

func (d *DomainSet) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(d.Type())),
		slog.Any("action", d.action),
	)
}

func NewDomainSet(rule *config.Rule, recorder *statistics.Recorder, target common.ActionTarget) *DomainSet {
	var a common.Action
	switch target {
	case common.ActionTargetHeader:
		a = action.NewHeaderAction(rule, recorder)
	case common.ActionTargetBody:
		a = action.NewBodyAction(rule, recorder)
	case common.ActionTargetURL:
		a = action.NewURLAction(rule, recorder)
	default:
		slog.Error("unknown target", "target", target)
		return nil
	}
	if a == nil {
		slog.Error("action.NewAction", "rule", rule)
		return nil
	}

	d := &DomainSet{
		action: a,
		source: rule.MatchValue,
	}

	// Load domain set asynchronously to avoid blocking
	go func() {
		slog.Info("loading domain set", "source", rule.MatchValue)
		if err := d.loadDomainSet(); err != nil {
			slog.Error("failed to load domain set", "source", rule.MatchValue, "error", err)
			return
		}
		slog.Info("domain set loaded", "source", rule.MatchValue, "count", len(d.domainSet))
	}()

	return d
}

// loadDomainSet loads domain list from source (local file or remote URL)
func (d *DomainSet) loadDomainSet() error {
	var data []byte
	var err error

	// Check if source is a URL or local file
	if strings.HasPrefix(d.source, "http://") || strings.HasPrefix(d.source, "https://") {
		// Load from URL
		data, err = d.loadFromURL(d.source)
	} else {
		// Load from local file
		data, err = os.ReadFile(d.source)
	}

	if err != nil {
		return err
	}

	// Parse domain list
	domains := d.parseDomainList(data)

	// Update domain set with lock
	d.mu.Lock()
	d.domainSet = domains
	d.loaded = true
	d.mu.Unlock()

	return nil
}

// loadFromURL downloads domain list from remote URL
func (d *DomainSet) loadFromURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	return io.ReadAll(resp.Body)
}

// parseDomainList parses domain list, ignoring lines starting with #
func (d *DomainSet) parseDomainList(data []byte) []string {
	var domains []string
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Add domain to list
		domains = append(domains, line)
	}

	return domains
}
