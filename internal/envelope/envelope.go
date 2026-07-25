package envelope

import (
	"fmt"
	"io"
	"time"

	"github.com/richhaase/c2/internal/jsonx"
)

type Envelope struct {
	Schema      string `json:"schema"`
	GeneratedAt string `json:"generated_at"`
	Data        any    `json:"data"`
}

func New(schema string, data any, now time.Time) Envelope {
	return Envelope{
		Schema:      schema,
		GeneratedAt: now.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		Data:        data,
	}
}

func Print(w io.Writer, schema string, data any) error {
	out, err := jsonx.Indent(New(schema, data, time.Now()))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(out))
	return err
}
