package invocationlog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cli-factory/internal/provider"
)

const Retention = 24 * time.Hour

type Log struct {
	ID             string           `json:"id"`
	Command        []string         `json:"command"`
	StartedAt      time.Time        `json:"started_at"`
	EndedAt        time.Time        `json:"ended_at"`
	DurationMillis int64            `json:"duration_millis"`
	Status         string           `json:"status"`
	ExitCode       int              `json:"exit_code"`
	Provider       string           `json:"provider,omitempty"`
	Tool           string           `json:"tool,omitempty"`
	Events         []provider.Event `json:"events,omitempty"`
	Result         any              `json:"result,omitempty"`
	Error          *provider.Error  `json:"error,omitempty"`
}

type Recorder struct {
	Log
	Dir  string
	Path string
}

func DefaultDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "cli-factory", "invocations"), nil
}

func Cleanup(dir string, now time.Time) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) > Retention {
			_ = os.Remove(path)
		}
	}
	return nil
}

func New(dir string, command []string) (*Recorder, error) {
	if dir == "" {
		var err error
		dir, err = DefaultDir()
		if err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	id := fmt.Sprintf("%s-%d", now.Format("20060102T150405.000000000Z"), os.Getpid())
	return &Recorder{
		Dir:  dir,
		Path: filepath.Join(dir, id+".json"),
		Log: Log{
			ID:        id,
			Command:   append([]string(nil), command...),
			StartedAt: now,
		},
	}, nil
}

func (r *Recorder) Emit(event provider.Event) {
	if event.Time == "" {
		event.Time = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if event.Provider == "" {
		event.Provider = r.Provider
	}
	if event.Tool == "" {
		event.Tool = r.Tool
	}
	r.Events = append(r.Events, event)
}

func (r *Recorder) Finish(status string, exitCode int, result any, errObj *provider.Error) error {
	r.EndedAt = time.Now().UTC()
	r.DurationMillis = r.EndedAt.Sub(r.StartedAt).Milliseconds()
	r.Status = status
	r.ExitCode = exitCode
	r.Result = result
	r.Error = errObj
	data, err := json.MarshalIndent(r.Log, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(r.Path, data, 0o600)
}
